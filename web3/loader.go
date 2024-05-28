package web3

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/gagliardetto/solana-go/programs/system"
	"log"
	"math"
	"strings"
	"sync"
	"time"
)

// CHUNK_SIZE Keep program chunks under PACKET_DATA_SIZE, leaving enough room for the
// rest of the Transaction fields
//
// TODO: replace 300 with a proper constant for the size of the other Transaction fields
const CHUNK_SIZE = PacketDataSize - 300

var Loader = LoaderImpl{}

// LoaderImpl Program loader interface
type LoaderImpl struct{}

func (impl LoaderImpl) ChunkSize() int {
	return CHUNK_SIZE
}

// GetMinNumSignatures calculates the minimum number of signatures required to load a program.
// It does not include retries and can be used to calculate transaction fees.
func (impl LoaderImpl) GetMinNumSignatures(dataLength int) int {
	return 2 * (int(math.Ceil(float64(dataLength)/float64(Loader.ChunkSize()))) + 1 + 1)
}

func (impl LoaderImpl) Load(connection Connection, payer Signer, program Signer, programId PublicKey, data []byte) (bool, error) {
	{
		balanceNeeded, err :=
			connection.GetMinimumBalanceForRentExemption(len(data), nil)
		if err != nil {
			return false, err
		}
		// Fetch program account info to check if it has already been created
		programInfo, err := connection.GetAccountInfo(program.PublicKey(), GetAccountInfoConfig{
			Commitment: &CommitmentConfirmed,
		})
		if err != nil {
			return false, err
		}

		var transaction *Transaction = nil

		tryNewTransaction := func() {
			if transaction == nil {
				transaction = &Transaction{}
			}
		}

		if programInfo != nil {
			if programInfo.Executable {
				log.Println("Program load failed, account is already executable")
				return false, nil
			}

			if len(programInfo.Data.Content) != len(data) {
				tryNewTransaction()
				err = transaction.AddInstruction3(system.NewAllocateInstruction(uint64(len(data)), program.PublicKey().D()).Build())
				if err != nil {
					return false, err
				}
			}

			if !programInfo.Owner.Equals(programId) {
				tryNewTransaction()
				err = transaction.AddInstruction3(system.NewAssignInstruction(program.PublicKey().D(), program.PublicKey().D()).Build())
				if err != nil {
					return false, err
				}
			}

			if programInfo.Lamports < balanceNeeded {
				tryNewTransaction()
				err = transaction.AddInstruction3(system.NewTransferInstruction(balanceNeeded-programInfo.Lamports, payer.PublicKey().D(), program.PublicKey().D()).Build())
				if err != nil {
					return false, err
				}
			}
		} else {
			transaction = &Transaction{}
			lamports := balanceNeeded
			if balanceNeeded <= 0 {
				balanceNeeded = 1
			}
			err = transaction.AddInstruction3(system.NewCreateAccountInstruction(
				lamports, uint64(len(data)), programId.D(), payer.PublicKey().D(), program.PublicKey().D(),
			).Build())
			if err != nil {
				return false, err
			}
		}

		// If the account is already created correctly, skip this step
		// and proceed directly to loading instructions
		if transaction != nil {
			_, err := connection.SendAndConfirmTransaction(context.Background(), *transaction, []Signer{payer, program}, ConfirmOptions{
				Commitment: &CommitmentConfirmed,
			})
			if err != nil {
				return false, err
			}
		}
	}

	var chunkSize = uint32(Loader.ChunkSize())
	var offset uint32 = 0
	var array = data
	var transactions []Transaction

	for len(array) > 0 {
		bs := array[:chunkSize]

		out := make([]byte, 0, int(chunkSize+16))
		out = binary.LittleEndian.AppendUint32(out, 0)      // instruction
		out = binary.LittleEndian.AppendUint32(out, offset) // offset
		out = binary.LittleEndian.AppendUint32(out, 0)      // bytesLength
		out = binary.LittleEndian.AppendUint32(out, 0)      // bytesLengthPadding
		out = append(out, bs[:]...)

		transaction := Transaction{}
		transaction.AddInstruction([]AccountMeta{
			{Pubkey: program.PublicKey(), IsSigner: true, IsWritable: true},
		}, programId, out)
		transactions = append(transactions, transaction)
		// Delay between sends in an attempt to reduce rate limit errors
		if strings.Contains(connection.rpcEndpoint, "solana.com") {
			time.Sleep((1000 / 4) * time.Millisecond)
		}

		offset += chunkSize
		array = array[chunkSize:]
	}

	if len(transactions) > 0 {
		wg := sync.WaitGroup{}
		wg.Add(len(transactions))
		for _, transaction := range transactions {
			transaction := transaction
			go func() {
				_, _ = connection.SendAndConfirmTransaction(context.Background(), transaction, []Signer{payer, program}, ConfirmOptions{
					Commitment: &CommitmentConfirmed,
				})
				wg.Done()
			}()
		}
		wg.Wait()
	}

	// Finalize the account loaded with program data for execution
	{
		transaction := Transaction{}
		transaction.AddInstruction([]AccountMeta{
			{
				Pubkey:     program.PublicKey(),
				IsSigner:   true,
				IsWritable: true,
			},
			{
				Pubkey:     SYSVAR_RENT_PUBKEY,
				IsSigner:   false,
				IsWritable: false,
			},
		}, programId, []byte{1, 0, 0, 0}) // Finalize instruction

		deployCommitment := CommitmentProcessed
		finalizeSignature, err := connection.SendTransaction(&transactions, []Signer{payer, program}, SendOptions{
			PreflightCommitment: &deployCommitment,
		})
		if err != nil {
			return false, err
		}
		resp, err := connection.ConfirmTransaction(context.Background(), BlockheightBasedTransactionConfirmationStrategy{
			Signature: finalizeSignature,
			BlockhashWithExpiryBlockHeight: BlockhashWithExpiryBlockHeight{
				LastValidBlockHeight: *transaction.LastValidBlockHeight,
				Blockhash:            *transaction.RecentBlockhash,
			},
		}, &deployCommitment)
		if err != nil {
			return false, err
		}
		if resp.Value.HasErr() {
			content, _ := json.Marshal(&resp.Value)
			return false, fmt.Errorf("transaction %s failed (%s)", finalizeSignature, string(content))
		}
		// We prevent programs from being usable until the slot after their deployment.
		// See https://github.com/solana-labs/solana/pull/29654
		for {
			currentSlot, err := connection.GetSlot(GetSlotConfig{
				Commitment: &deployCommitment,
			})
			if err == nil {
				if currentSlot > resp.Context.Slot {
					break
				}
			}
			time.Sleep((MS_PER_SLOT / 2) * time.Millisecond)
		}
	}

	// success
	return true, nil
}
