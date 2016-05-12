package frames

import (
	"bytes"
	"errors"

	"github.com/lucas-clemente/quic-go/protocol"
	"github.com/lucas-clemente/quic-go/utils"
)

// A StopWaitingFrame in QUIC
type StopWaitingFrame struct {
	LeastUnacked    protocol.PacketNumber
	Entropy         byte
	PacketNumberLen protocol.PacketNumberLen
	PacketNumber    protocol.PacketNumber
}

var (
	errLeastUnackedHigherThanPacketNumber = errors.New("StopWaitingFrame: LeastUnacked can't be greater than the packet number")
	errPacketNumberNotSet                 = errors.New("StopWaitingFrame: PacketNumber not set")
	errPacketNumberLenNotSet              = errors.New("StopWaitingFrame: PacketNumberLen not set")
)

func (f *StopWaitingFrame) Write(b *bytes.Buffer, version protocol.VersionNumber) error {
	// packetNumber is the packet number of the packet that this StopWaitingFrame will be sent with
	typeByte := uint8(0x06)
	b.WriteByte(typeByte)

	b.WriteByte(f.Entropy)

	// make sure the PacketNumber was set
	if f.PacketNumber == protocol.PacketNumber(0) {
		return errPacketNumberNotSet
	}

	if f.LeastUnacked > f.PacketNumber {
		return errLeastUnackedHigherThanPacketNumber
	}

	leastUnackedDelta := uint64(f.PacketNumber - f.LeastUnacked)

	switch f.PacketNumberLen {
	case protocol.PacketNumberLen1:
		b.WriteByte(uint8(leastUnackedDelta))
	case protocol.PacketNumberLen2:
		utils.WriteUint16(b, uint16(leastUnackedDelta))
	case protocol.PacketNumberLen4:
		utils.WriteUint32(b, uint32(leastUnackedDelta))
	case protocol.PacketNumberLen6:
		utils.WriteUint48(b, leastUnackedDelta)
	default:
		return errPacketNumberLenNotSet
	}

	return nil
}

// MinLength of a written frame
func (f *StopWaitingFrame) MinLength() (protocol.ByteCount, error) {
	if f.PacketNumberLen == protocol.PacketNumberLenInvalid {
		return 0, errPacketNumberLenNotSet
	}
	return protocol.ByteCount(1 + 1 + f.PacketNumberLen), nil
}

// ParseStopWaitingFrame parses a StopWaiting frame
func ParseStopWaitingFrame(r *bytes.Reader, packetNumber protocol.PacketNumber, packetNumberLen protocol.PacketNumberLen) (*StopWaitingFrame, error) {
	frame := &StopWaitingFrame{}

	// read the TypeByte
	_, err := r.ReadByte()
	if err != nil {
		return nil, err
	}

	frame.Entropy, err = r.ReadByte()
	if err != nil {
		return nil, err
	}

	leastUnackedDelta, err := utils.ReadUintN(r, uint8(packetNumberLen))
	if err != nil {
		return nil, err
	}

	if leastUnackedDelta > uint64(packetNumber) {
		return nil, errors.New("StopWaitingFrame: Invalid LeastUnackedDelta")
	}

	frame.LeastUnacked = protocol.PacketNumber(uint64(packetNumber) - leastUnackedDelta)

	return frame, nil
}
