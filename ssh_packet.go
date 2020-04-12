package tetris

import (
	"encoding/binary"
	"io"

	"golang.org/x/xerrors"
)

// Packet is a message
type Packet struct {
	Data []byte
}

// Write writes binary that marshalled from packet to io.Writer
func (p *Packet) Write(w io.Writer) error {
	header := make([]byte, 4) // TODO: define protocol

	binary.BigEndian.PutUint32(header, uint32(len(p.Data)))
	if _, err := w.Write(header); err != nil {
		return xerrors.Errorf("failed to Write header: %w", err)
	}
	if _, err := w.Write(p.Data); err != nil {
		return xerrors.Errorf("failed to Write data: %w", err)
	}
	return nil
}

// ReadPacker reads Packet data from Reader
func ReadPacket(r io.Reader) (*Packet, error) {
	header := make([]byte, 4) // TODO: define protocol
	if _, err := r.Read(header); err != nil {
		return nil, xerrors.Errorf("failed to read header: %w", err)
	}

	len := binary.BigEndian.Uint32(header)
	buf := make([]byte, len)
	if _, err := r.Read(buf); err != nil {
		return nil, xerrors.Errorf("failed to read data: %w", err)
	}

	return &Packet{
		Data: buf,
	}, nil
}
