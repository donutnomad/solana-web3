package web3

import (
	"github.com/donutnomad/solana-web3/web3/utils"
)

const MINIMUM_SLOT_PER_EPOCH = 32

// EpochSchedule represents the epoch schedule.
type EpochSchedule struct {
	// The maximum number of slots in each epoch
	SlotsPerEpoch int `json:"slotsPerEpoch,omitempty"`
	// The number of slots before the beginning of an epoch to calculate a leader schedule for that epoch
	LeaderScheduleSlotOffset int `json:"leaderScheduleSlotOffset,omitempty"`
	// Indicates whether epochs start short and grow
	Warmup bool `json:"warmup,omitempty"`
	// The first epoch with `SlotsPerEpoch` slots
	FirstNormalEpoch int `json:"firstNormalEpoch,omitempty"`
	// The first slot of `FirstNormalEpoch`
	FirstNormalSlot int `json:"firstNormalSlot,omitempty"`
}

func NewEpochSchedule(slotsPerEpoch, leaderScheduleSlotOffset int, warmup bool, firstNormalEpoch, firstNormalSlot int) *EpochSchedule {
	return &EpochSchedule{
		SlotsPerEpoch:            slotsPerEpoch,
		LeaderScheduleSlotOffset: leaderScheduleSlotOffset,
		Warmup:                   warmup,
		FirstNormalEpoch:         firstNormalEpoch,
		FirstNormalSlot:          firstNormalSlot,
	}
}

func (es *EpochSchedule) GetEpoch(slot int) int {
	a, _ := es.GetEpochAndSlotIndex(slot)
	return a
}

func (es *EpochSchedule) GetEpochAndSlotIndex(slot int) (int, int) {
	if slot < es.FirstNormalSlot {
		epoch := trailingZeros(nextPowerOfTwo(slot+MINIMUM_SLOT_PER_EPOCH+1)) -
			trailingZeros(MINIMUM_SLOT_PER_EPOCH) - 1

		epochLen := es.GetSlotsInEpoch(epoch)
		slotIndex := slot - (epochLen - MINIMUM_SLOT_PER_EPOCH)
		return epoch, slotIndex
	} else {
		normalSlotIndex := slot - es.FirstNormalSlot
		normalEpochIndex := normalSlotIndex / es.SlotsPerEpoch
		epoch := es.FirstNormalEpoch + normalEpochIndex
		slotIndex := normalSlotIndex % es.SlotsPerEpoch
		return epoch, slotIndex
	}
}

func (es *EpochSchedule) GetFirstSlotInEpoch(epoch int) int {
	if epoch <= es.FirstNormalEpoch {
		return (utils.PowInt(2, epoch) - 1) * MINIMUM_SLOT_PER_EPOCH
	} else {
		return (epoch-es.FirstNormalEpoch)*es.SlotsPerEpoch + es.FirstNormalSlot
	}
}

func (es *EpochSchedule) GetLastSlotInEpoch(epoch int) int {
	return es.GetFirstSlotInEpoch(epoch) + es.GetSlotsInEpoch(epoch) - 1
}

func (es *EpochSchedule) GetSlotsInEpoch(epoch int) int {
	if epoch < es.FirstNormalEpoch {
		return utils.PowInt(2, epoch+trailingZeros(MINIMUM_SLOT_PER_EPOCH))
	} else {
		return es.SlotsPerEpoch
	}
}

func trailingZeros(n int) int {
	trailingZeros := 0
	for n > 1 {
		n /= 2
		trailingZeros++
	}
	return trailingZeros
}

func nextPowerOfTwo(n int) int {
	if n == 0 {
		return 1
	}
	n--
	n |= n >> 1
	n |= n >> 2
	n |= n >> 4
	n |= n >> 8
	n |= n >> 16
	n |= n >> 32
	return n + 1
}
