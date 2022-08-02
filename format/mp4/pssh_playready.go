package mp4

import (
	"github.com/wader/fq/format"
	"github.com/wader/fq/pkg/decode"
	"github.com/wader/fq/pkg/interp"
	"github.com/wader/fq/pkg/scalar"
)

func init() {
	interp.RegisterFormat(decode.Format{
		Name:        format.PSSH_PLAYREADY,
		Description: "PlayReady PSSH",
		DecodeFn:    playreadyPsshDecode,
	})
}

const (
	recordTypeRightsManagementHeader = 1
	recordTypeLicenseStore           = 2
)

var recordTypeNames = scalar.UToSymStr{
	recordTypeRightsManagementHeader: "Rights management header",
	recordTypeLicenseStore:           "License store",
}

func playreadyPsshDecode(d *decode.D, _ any) any {
	d.Endian = decode.LittleEndian

	d.FieldU32("size")
	count := d.FieldU16("count")
	i := uint64(0)
	d.FieldStructArrayLoop("records", "record", func() bool { return i < count }, func(d *decode.D) {
		recordType := d.FieldU16("type", recordTypeNames)
		recordLen := d.FieldU16("len")
		switch recordType {
		case recordTypeRightsManagementHeader, recordTypeLicenseStore:
			d.FieldUTF16LE("xml", int(recordLen))
		default:
			d.FieldRawLen("data", int64(recordLen)*8)
		}
		i++
	})

	return nil
}
