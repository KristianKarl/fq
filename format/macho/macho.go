package macho

// https://github.com/aidansteele/osx-abi-macho-file-format-reference

import (
	"embed"
	"time"

	"github.com/wader/fq/format"
	"github.com/wader/fq/pkg/bitio"
	"github.com/wader/fq/pkg/decode"
	"github.com/wader/fq/pkg/interp"
	"github.com/wader/fq/pkg/scalar"
)

//go:embed macho.jq
var machoFS embed.FS

func init() {
	interp.RegisterFormat(decode.Format{
		Name:        format.MACHO,
		Description: "Mach-O macOS executable",
		Groups:      []string{format.PROBE},
		DecodeFn:    machoDecode,
		Functions:   []string{"_help"},
	})
	interp.RegisterFS(machoFS)
}

//nolint:revive
const (
	MH_MAGIC    = 0xfeed_face
	MH_CIGAM    = 0xcefa_edfe
	MH_MAGIC_64 = 0xfeed_facf
	MH_CIGAM_64 = 0xcffa_edfe
	FAT_MAGIC   = 0xcafe_babe
	FAT_CIGAM   = 0xbeba_feca
)

var magicSymMapper = scalar.UToDescription{
	MH_MAGIC:    "32-bit little endian",
	MH_CIGAM:    "32-bit big endian",
	MH_MAGIC_64: "64-bit little endian",
	MH_CIGAM_64: "64-bit big endian",
}

var endianNames = scalar.UToSymStr{
	MH_MAGIC:    "little_endian",
	MH_CIGAM:    "big_endian",
	MH_MAGIC_64: "little_endian",
	MH_CIGAM_64: "big_endian",
}

var cpuTypes = scalar.UToSymStr{
	0xff_ff_ff_ff: "any",
	1:             "vax",
	2:             "romp",
	4:             "ns32032",
	5:             "ns32332",
	6:             "mc680x0",
	7:             "x86",
	8:             "mips",
	9:             "ns32532",
	10:            "mc98000",
	11:            "hppa",
	12:            "arm",
	13:            "mc88000",
	14:            "sparc",
	15:            "i860",
	16:            "i860_little",
	17:            "rs6000",
	18:            "powerpc",
	0x1000007:     "x86_64",
	0x100000c:     "arm64",
	0x1000013:     "powerpc64",
	255:           "veo",
}

func intelSubTypeHelper(f, m uint64) uint64 {
	return f + (m << 4)
}

var cpuSubTypes = map[uint64]scalar.UToSymStr{
	0xff_ff_ff_ff: {
		0xff_ff_ff_ff: "multiple",
	},
	1: {
		0xff_ff_ff_ff: "multiple",
		0:             "vax_all",
		1:             "vax780",
		2:             "vax785",
		3:             "vax750",
		4:             "vax730",
		5:             "uvaxi",
		6:             "uvaxii",
		7:             "vax8200",
		8:             "vax8500",
		9:             "vax8600",
		10:            "vax8650",
		11:            "vax8800",
		12:            "uvaxiii",
	},
	6: {
		0xff_ff_ff_ff: "multiple",
		1:             "mc680x0_all", // 1: mc68030
		2:             "mc68040",
		3:             "mc68030_only",
	},
	7: {
		0xff_ff_ff_ff:             "multiple",
		intelSubTypeHelper(3, 0):  "i386_all", // i386
		intelSubTypeHelper(4, 0):  "i486",
		intelSubTypeHelper(4, 8):  "486sx",
		intelSubTypeHelper(5, 0):  "pent",
		intelSubTypeHelper(6, 1):  "pentpro",
		intelSubTypeHelper(6, 3):  "pentii_m3",
		intelSubTypeHelper(6, 5):  "pentii_m5",
		intelSubTypeHelper(7, 6):  "celeron",
		intelSubTypeHelper(7, 7):  "celeron_mobile",
		intelSubTypeHelper(8, 0):  "pentium_3",
		intelSubTypeHelper(8, 1):  "pentium_3_m",
		intelSubTypeHelper(8, 2):  "pentium_3_xeon",
		intelSubTypeHelper(9, 0):  "pentium_m",
		intelSubTypeHelper(10, 0): "pentium_4",
		intelSubTypeHelper(10, 1): "pentium_4_m",
		intelSubTypeHelper(11, 0): "itanium",
		intelSubTypeHelper(11, 1): "itanium_2",
		intelSubTypeHelper(12, 0): "xeon",
		intelSubTypeHelper(12, 1): "xeon_2",
	},
	8: {
		0xff_ff_ff_ff: "multiple",
		0:             "mips_all",
		1:             "mips_r2300",
		2:             "mips_r2600",
		3:             "mips_r2800",
		4:             "mips_r2000a",
		5:             "mips_r2000",
		6:             "mips_r3000a",
		7:             "mips_r3000",
	},
	10: {
		0xff_ff_ff_ff: "multiple",
		0:             "mc98000_all",
		1:             "mc98001",
	},
	11: {
		0xff_ff_ff_ff: "multiple",
		0:             "hppa_all",
		1:             "hppa_7100",
		2:             "hppa_7100_lc",
	},
	12: {
		0xff_ff_ff_ff: "multiple",
		0:             "arm_all",
		5:             "arm_v4t",
		6:             "arm_v6",
		7:             "arm_v5tej",
		8:             "arm_xscale",
		9:             "arm_v7",
		10:            "arm_v7f",
		11:            "arm_v7s",
		12:            "arm_v7k",
		13:            "arm_v8",
		14:            "arm_v6m",
		15:            "arm_v7m",
		16:            "arm_v7em",
	},
	13: {
		0xff_ff_ff_ff: "multiple",
		0:             "mc88000_all",
		1:             "mc88100",
		2:             "mc88110",
	},
	14: {
		0xff_ff_ff_ff: "multiple",
		0:             "sparc_all",
	},
	15: {
		0xff_ff_ff_ff: "multiple",
		0:             "i860_all",
		1:             "i860_a860",
	},
	18: {
		0xff_ff_ff_ff: "multiple",
		0:             "powerpc_all",
		1:             "powerpc_601",
		2:             "powerpc_602",
		3:             "powerpc_603",
		4:             "powerpc_603e",
		5:             "powerpc_603ev",
		6:             "powerpc_604",
		7:             "powerpc_604e",
		8:             "powerpc_620",
		9:             "powerpc_750",
		10:            "powerpc_7400",
		11:            "powerpc_7450",
		100:           "powerpc_970",
	},
	0x1000012: {
		0xff_ff_ff_ff: "multiple",
		0:             "arm64_all",
		1:             "arm64_v8",
		2:             "arm64_e",
	},
}

var fileTypes = scalar.UToSymStr{
	0x1: "object",
	0x2: "execute",
	0x3: "fvmlib",
	0x4: "core",
	0x5: "preload",
	0x6: "dylib",
	0x7: "dylinker",
	0x8: "bundle",
	0x9: "dylib_stub",
	0xa: "dsym",
	0xb: "kext_bundle",
}

//nolint:revive
const (
	LC_REQ_DYLD                 = 0x80000000
	LC_SEGMENT                  = 0x1
	LC_SYMTAB                   = 0x2
	LC_SYMSEG                   = 0x3
	LC_THREAD                   = 0x4
	LC_UNIXTHREAD               = 0x5
	LC_LOADFVMLIB               = 0x6
	LC_IDFVMLIB                 = 0x7
	LC_IDENT                    = 0x8 // not implemented
	LC_FVMFILE                  = 0x9 // not implemented
	LC_PREPAGE                  = 0xa // not implemented
	LC_DYSYMTAB                 = 0xb
	LC_LOAD_DYLIB               = 0xc
	LC_ID_DYLIB                 = 0xd
	LC_LOAD_DYLINKER            = 0xe
	LC_ID_DYLINKER              = 0xf
	LC_PREBOUND_DYLIB           = 0x10
	LC_ROUTINES                 = 0x11
	LC_SUB_FRAMEWORK            = 0x12
	LC_SUB_UMBRELLA             = 0x13
	LC_SUB_CLIENT               = 0x14
	LC_SUB_LIBRARY              = 0x15
	LC_TWOLEVEL_HINTS           = 0x16
	LC_PREBIND_CKSUM            = 0x17 // not implemented
	LC_LOAD_WEAK_DYLIB          = 0x80000018
	LC_SEGMENT_64               = 0x19
	LC_ROUTINES_64              = 0x1a
	LC_UUID                     = 0x1b
	LC_RPATH                    = 0x8000001c
	LC_CODE_SIGNATURE           = 0x1d
	LC_SEGMENT_SPLIT_INFO       = 0x1e
	LC_REEXPORT_DYLIB           = 0x8000001f
	LC_LAZY_LOAD_DYLIB          = 0x20
	LC_ENCRYPTION_INFO          = 0x21
	LC_DYLD_INFO                = 0x22
	LC_DYLD_INFO_ONLY           = 0x80000022
	LC_LOAD_UPWARD_DYLIB        = 0x80000023
	LC_VERSION_MIN_MACOSX       = 0x24
	LC_VERSION_MIN_IPHONEOS     = 0x25
	LC_FUNCTION_STARTS          = 0x26
	LC_DYLD_ENVIRONMENT         = 0x27
	LC_MAIN                     = 0x80000028
	LC_DATA_IN_CODE             = 0x29
	LC_SOURCE_VERSION           = 0x2a
	LC_DYLIB_CODE_SIGN_DRS      = 0x2b
	LC_ENCRYPTION_INFO_64       = 0x2c
	LC_LINKER_OPTION            = 0x2d
	LC_LINKER_OPTIMIZATION_HINT = 0x2e
	LC_VERSION_MIN_TVOS         = 0x2f
	LC_VERSION_MIN_WATCHOS      = 0x30
	LC_NOTE                     = 0x31 // not implemented
	LC_BUILD_VERSION            = 0x32
)

var loadCommands = scalar.UToSymStr{
	LC_REQ_DYLD:                 "req_dyld",
	LC_SEGMENT:                  "segment",
	LC_SYMTAB:                   "symtab",
	LC_SYMSEG:                   "symseg",
	LC_THREAD:                   "thread",
	LC_UNIXTHREAD:               "unixthread",
	LC_LOADFVMLIB:               "loadfvmlib",
	LC_IDFVMLIB:                 "idfvmlib",
	LC_IDENT:                    "ident",
	LC_FVMFILE:                  "fvmfile",
	LC_PREPAGE:                  "prepage",
	LC_DYSYMTAB:                 "dysymtab",
	LC_LOAD_DYLIB:               "load_dylib",
	LC_ID_DYLIB:                 "id_dylib",
	LC_LOAD_DYLINKER:            "load_dylinker",
	LC_ID_DYLINKER:              "id_dylinker",
	LC_PREBOUND_DYLIB:           "prebound_dylib",
	LC_ROUTINES:                 "routines",
	LC_SUB_FRAMEWORK:            "sub_framework",
	LC_SUB_UMBRELLA:             "sub_umbrella",
	LC_SUB_CLIENT:               "sub_client",
	LC_SUB_LIBRARY:              "sub_library",
	LC_TWOLEVEL_HINTS:           "twolevel_hints",
	LC_PREBIND_CKSUM:            "prebind_cksum",
	LC_LOAD_WEAK_DYLIB:          "load_weak_dylib",
	LC_SEGMENT_64:               "segment_64",
	LC_ROUTINES_64:              "routines_64",
	LC_UUID:                     "uuid",
	LC_RPATH:                    "rpath",
	LC_CODE_SIGNATURE:           "code_signature",
	LC_SEGMENT_SPLIT_INFO:       "segment_split_info",
	LC_REEXPORT_DYLIB:           "reexport_dylib",
	LC_LAZY_LOAD_DYLIB:          "lazy_load_dylib",
	LC_ENCRYPTION_INFO:          "encryption_info",
	LC_DYLD_INFO:                "dyld_info",
	LC_DYLD_INFO_ONLY:           "dyld_info_only",
	LC_LOAD_UPWARD_DYLIB:        "load_upward_dylib",
	LC_VERSION_MIN_MACOSX:       "version_min_macosx",
	LC_VERSION_MIN_IPHONEOS:     "version_min_iphoneos",
	LC_FUNCTION_STARTS:          "function_starts",
	LC_DYLD_ENVIRONMENT:         "dyld_environment",
	LC_MAIN:                     "main",
	LC_DATA_IN_CODE:             "data_in_code",
	LC_SOURCE_VERSION:           "source_version",
	LC_DYLIB_CODE_SIGN_DRS:      "dylib_code_sign_drs",
	LC_ENCRYPTION_INFO_64:       "encryption_info_64",
	LC_LINKER_OPTION:            "linker_option",
	LC_LINKER_OPTIMIZATION_HINT: "linker_optimization_hint",
	LC_VERSION_MIN_TVOS:         "version_min_tvos",
	LC_VERSION_MIN_WATCHOS:      "version_min_watchos",
	LC_NOTE:                     "note",
	LC_BUILD_VERSION:            "build_version",
}

var sectionTypes = scalar.UToSymStr{
	0x0:  "regular",
	0x1:  "zerofill",
	0x2:  "cstring_literals",
	0x3:  "4byte_literals",
	0x4:  "8byte_literals",
	0x5:  "literal_pointers",
	0x6:  "non_lazy_symbol_pointers",
	0x7:  "lazy_symbol_pointers",
	0x8:  "symbol_stubs",
	0x9:  "mod_init_func_pointers",
	0xa:  "mod_term_func_pointers",
	0xb:  "coalesced",
	0xc:  "gb_zerofill",
	0xd:  "interposing",
	0xe:  "16byte_literals",
	0xf:  "dtrace_dof",
	0x10: "lazy_dylib_symbol_pointers",
	0x11: "thread_local_regular",
	0x12: "thread_local_zerofill",
	0x13: "thread_local_variables",
	0x14: "thread_local_variable_pointers",
	0x15: "thread_local_init_function_pointers",
}

func machoDecode(d *decode.D, _ any) any {
	ofileDecode(d)
	return nil
}

func ofileDecode(d *decode.D) {
	var archBits int
	var cpuType uint64
	var ncmds uint64
	magicBuffer := d.U32LE()

	if magicBuffer == MH_MAGIC || magicBuffer == MH_MAGIC_64 {
		d.Endian = decode.LittleEndian
		if magicBuffer == MH_MAGIC {
			archBits = 32
		} else {
			archBits = 64
		}
	} else if magicBuffer == MH_CIGAM || magicBuffer == MH_CIGAM_64 {
		d.Endian = decode.BigEndian
		if magicBuffer == MH_CIGAM {
			archBits = 32
		} else {
			archBits = 64
		}
	} else if magicBuffer == FAT_MAGIC {
		d.Endian = decode.LittleEndian
		fatParse(d)
		return
	} else if magicBuffer == FAT_CIGAM {
		d.Endian = decode.BigEndian
		fatParse(d)
		return
	} else {
		// AR files are also valid OFiles but they should be parsed by `-d ar`
		d.Fatalf("Invalid magic field")
	}

	d.SeekRel(-4 * 8)
	d.FieldStruct("header", func(d *decode.D) {
		d.FieldValueS("arch_bits", int64(archBits))
		magic := d.FieldU32("magic", magicSymMapper, scalar.ActualHex)
		d.FieldValueU("bits", uint64(archBits))
		d.FieldValueStr("endian", endianNames[magic])
		cpuType = d.FieldU32("cputype", cpuTypes, scalar.ActualHex)
		d.FieldU32("cpusubtype", cpuSubTypes[cpuType], scalar.ActualHex)
		d.FieldU32("filetype", fileTypes)
		ncmds = d.FieldU32("ncdms")
		d.FieldU32("sizeofncdms")
		d.FieldStruct("flags", parseMachHeaderFlags)
		if archBits == 64 {
			d.FieldRawLen("reserved", 4*8, d.BitBufIsZero())
		}
	})
	d.FieldArray("load_commands", func(d *decode.D) {
		for i := uint64(0); i < ncmds; i++ {
			d.FieldStruct("load_command", func(d *decode.D) {
				cmd := d.FieldU32("cmd", loadCommands, scalar.ActualHex)
				cmdsize := d.FieldU32("cmdsize")
				switch cmd {
				case LC_UUID:
					d.FieldStruct("uuid_command", func(d *decode.D) {
						d.FieldRawLen("uuid", 16*8)
					})
				case LC_SEGMENT, LC_SEGMENT_64:
					// nsect := (cmdsize - uint64(archBits)) / uint64(archBits)
					var nsects uint64
					d.FieldStruct("segment_command", func(d *decode.D) {
						d.FieldValueS("arch_bits", int64(archBits))
						d.FieldUTF8NullFixedLen("segname", 16) // OPCODE_DECODER segname==__TEXT
						if archBits == 32 {
							d.FieldU32("vmaddr", scalar.ActualHex)
							d.FieldU32("vmsize")
							d.FieldU32("fileoff")
							d.FieldU32("tfilesize")
						} else {
							d.FieldU64("vmaddr", scalar.ActualHex)
							d.FieldU64("vmsize")
							d.FieldU64("fileoff")
							d.FieldU64("tfilesize")
						}
						d.FieldS32("initprot")
						d.FieldS32("maxprot")
						nsects = d.FieldU32("nsects")
						d.FieldStruct("flags", parseSegmentFlags)
					})
					d.FieldArray("sections", func(d *decode.D) {
						for i := uint64(0); i < nsects; i++ {
							d.FieldStruct("section", func(d *decode.D) {
								// OPCODE_DECODER sectname==__text
								d.FieldUTF8NullFixedLen("sectname", 16)
								d.FieldUTF8NullFixedLen("segname", 16)
								var size uint64
								if archBits == 32 {
									d.FieldU32("address", scalar.ActualHex)
									size = d.FieldU32("size")
								} else {
									d.FieldU64("address", scalar.ActualHex)
									size = d.FieldU64("size")
								}
								offset := d.FieldU32("offset")
								d.FieldU32("align")
								d.FieldU32("reloff")
								d.FieldU32("nreloc")
								// get section type
								d.FieldStruct("flags", parseSectionFlags)
								d.FieldU8("type", sectionTypes)
								d.FieldU32("reserved1")
								d.FieldU32("reserved2")
								if archBits == 64 {
									d.FieldU32("reserved3")
								}
								d.RangeFn(int64(offset)*8, int64(size)*8, func(d *decode.D) {
									d.FieldRawLen("data", d.BitsLeft())
								})
							})
						}
					})
				case LC_TWOLEVEL_HINTS:
					d.FieldU32("offset")
					d.FieldU32("nhints")
				case LC_LOAD_DYLIB, LC_ID_DYLIB, LC_LOAD_UPWARD_DYLIB, LC_LOAD_WEAK_DYLIB, LC_LAZY_LOAD_DYLIB, LC_REEXPORT_DYLIB:
					d.FieldStruct("dylib_command", func(d *decode.D) {
						offset := d.FieldU32("offset")
						d.FieldU32("timestamp", timestampMapper)
						d.FieldU32("current_version")
						d.FieldU32("compatibility_version")
						d.FieldUTF8NullFixedLen("name", int(cmdsize)-int(offset))
					})
				case LC_LOAD_DYLINKER, LC_ID_DYLINKER, LC_DYLD_ENVIRONMENT:
					offset := d.FieldU32("offset")
					d.FieldUTF8NullFixedLen("name", int(cmdsize)-int(offset))
				case LC_RPATH:
					offset := d.FieldU32("offset")
					d.FieldUTF8NullFixedLen("name", int(cmdsize)-int(offset))
				case LC_PREBOUND_DYLIB:
					// https://github.com/aidansteele/osx-abi-macho-file-format-reference#prebound_dylib_command
					d.U32() // name_offset
					nmodules := d.FieldU32("nmodules")
					d.U32() // linked_modules_offset
					d.FieldUTF8Null("name")
					d.FieldBitBufFn("linked_modules", func(d *decode.D) bitio.ReaderAtSeeker {
						return d.RawLen(int64((nmodules / 8) + (nmodules % 8)))
					})
				case LC_THREAD, LC_UNIXTHREAD:
					d.FieldU32("flavor")
					count := d.FieldU32("count")
					d.FieldStruct("state", func(d *decode.D) {
						switch cpuType {
						case 0x7:
							threadStateI386Decode(d)
						case 0xC:
							threadStateARM32Decode(d)
						case 0x13:
							threadStatePPC32Decode(d)
						case 0x1000007:
							threadStateX8664Decode(d)
						case 0x100000C:
							threadStateARM64Decode(d)
						case 0x1000013:
							threadStatePPC64Decode(d)
						default:
							d.FieldRawLen("state", int64(count*32))
						}
					})
				case LC_ROUTINES, LC_ROUTINES_64:
					if archBits == 32 {
						d.FieldU32("init_address", scalar.ActualHex)
						d.FieldU32("init_module")
						d.FieldU32("reserved1")
						d.FieldU32("reserved2")
						d.FieldU32("reserved3")
						d.FieldU32("reserved4")
						d.FieldU32("reserved5")
						d.FieldU32("reserved6")
					} else {
						d.FieldU64("init_address", scalar.ActualHex)
						d.FieldU64("init_module")
						d.FieldU64("reserved1")
						d.FieldU64("reserved2")
						d.FieldU64("reserved3")
						d.FieldU64("reserved4")
						d.FieldU64("reserved5")
						d.FieldU64("reserved6")
					}
				case LC_SUB_UMBRELLA, LC_SUB_LIBRARY, LC_SUB_CLIENT, LC_SUB_FRAMEWORK:
					offset := d.FieldU32("offset")
					d.FieldUTF8NullFixedLen("name", int(cmdsize)-int(offset))
				case LC_SYMTAB:
					d.FieldU32("symoff")
					d.FieldU32("nsyms")
					d.FieldU32("stroff")
					d.FieldU32("strsize")
				case LC_DYSYMTAB:
					d.FieldU32("ilocalsym")
					d.FieldU32("nlocalsym")
					d.FieldU32("iextdefsym")
					d.FieldU32("nextdefsym")
					d.FieldU32("iundefsym")
					d.FieldU32("nundefsym")
					d.FieldU32("tocoff")
					d.FieldU32("ntoc")
					d.FieldU32("modtaboff")
					d.FieldU32("nmodtab")
					d.FieldU32("extrefsymoff")
					d.FieldU32("nextrefsyms")
					d.FieldU32("indirectsymoff")
					d.FieldU32("nindirectsyms")

					d.FieldU32("extreloff")
					d.FieldU32("nextrel")
					d.FieldU32("locreloff")
					d.FieldU32("nlocrel")
				case LC_BUILD_VERSION:
					d.FieldU32("platform")
					d.FieldU32("minos")
					d.FieldU32("sdk")
					ntools := d.FieldU32("ntools")
					var ntoolsIdx uint64
					d.FieldStructArrayLoop("tools", "tool", func() bool {
						return ntoolsIdx < ntools
					}, func(d *decode.D) {
						d.FieldU32("tool")
						d.FieldU32("version")
						ntoolsIdx++
					})
				case LC_CODE_SIGNATURE, LC_SEGMENT_SPLIT_INFO, LC_FUNCTION_STARTS, LC_DATA_IN_CODE, LC_DYLIB_CODE_SIGN_DRS, LC_LINKER_OPTIMIZATION_HINT:
					d.FieldStruct("linkedit_data", func(d *decode.D) {
						d.FieldU32("off")
						d.FieldU32("size")
					})
				case LC_VERSION_MIN_IPHONEOS, LC_VERSION_MIN_MACOSX, LC_VERSION_MIN_TVOS, LC_VERSION_MIN_WATCHOS:
					d.FieldU32("version")
					d.FieldU32("sdk")
				case LC_DYLD_INFO, LC_DYLD_INFO_ONLY:
					d.FieldStruct("dyld_info", func(d *decode.D) {
						d.FieldU32("rebase_off")
						d.FieldU32("rebase_size")
						d.FieldU32("bind_off")
						d.FieldU32("bind_size")
						d.FieldU32("weak_bind_off")
						d.FieldU32("weak_bind_size")
						d.FieldU32("lazy_bind_off")
						d.FieldU32("lazy_bind_size")
						d.FieldU32("export_off")
						d.FieldU32("export_size")
					})
				case LC_MAIN:
					d.FieldStruct("entrypoint", func(d *decode.D) {
						d.FieldU64("entryoff")
						d.FieldU64("stacksize")
					})
				case LC_SOURCE_VERSION:
					d.FieldStruct("source_version_tag", func(d *decode.D) {
						d.FieldU64("tag")
					})
				case LC_LINKER_OPTION:
					d.FieldStruct("linker_option", func(d *decode.D) {
						count := d.FieldU32("count")
						d.FieldUTF8NullFixedLen("option", int(count))
					})
				case LC_ENCRYPTION_INFO, LC_ENCRYPTION_INFO_64:
					d.FieldStruct("encryption_info", func(d *decode.D) {
						offset := d.FieldU32("offset")
						size := d.FieldU32("size")
						d.FieldU32("id")
						d.RangeFn(int64(offset)*8, int64(size)*8, func(d *decode.D) {
							d.FieldRawLen("data", d.BitsLeft())
						})
					})
				case LC_IDFVMLIB, LC_LOADFVMLIB:
					d.FieldStruct("fvmlib", func(d *decode.D) {
						offset := d.FieldU32("offset")
						d.FieldU32("minor_version")
						d.FieldU32("header_addr", scalar.ActualHex)
						d.FieldUTF8NullFixedLen("name", int(cmdsize)-int(offset))
					})
				default:
					if _, ok := loadCommands[cmd]; !ok {
						d.SeekRel(int64((cmdsize - 8) * 8))
						// Seek Rel so the parts are marked unknown
					}
				}
			})
		}
	})
}

func fatParse(d *decode.D) {
	// Go to start of the file again
	d.SeekAbs(0)
	var narchs uint64
	var ofileOffsets []uint64
	d.FieldStruct("fat_header", func(d *decode.D) {
		d.FieldU32("magic", scalar.ActualHex)
		narchs = d.FieldU32("narchs")
		narchsIdx := 0

		d.FieldStructArrayLoop("archs", "fat_arch", func() bool {
			return narchsIdx < int(narchs)
		}, func(d *decode.D) {
			// parse FatArch
			// beware cputype and cpusubtype changes from ofile header to fat header
			cpuType := d.FieldU32("cputype", cpuTypes, scalar.ActualHex)
			d.FieldU32("cpusubtype", cpuSubTypes[cpuType], scalar.ActualHex)
			ofileOffsets = append(ofileOffsets, d.FieldU32("offset"))
			d.FieldU32("size")
			d.FieldU32("align")
			narchsIdx++
		})
	})
	nfilesIdx := 0
	d.FieldStructArrayLoop("files", "file", func() bool {
		return nfilesIdx < int(narchs)
	}, func(d *decode.D) {
		d.SeekAbs(int64(ofileOffsets[nfilesIdx]) * 8)
		ofileDecode(d)
		nfilesIdx++
	})
}

func parseMachHeaderFlags(d *decode.D) {
	d.FieldRawLen("reserved", 6)
	d.FieldBool("app_extension_safe")
	d.FieldBool("no_heap_execution")

	d.FieldBool("has_tlv_descriptors")
	d.FieldBool("dead_strippable_dylib")
	d.FieldBool("pie")
	d.FieldBool("no_reexported_dylibs")

	d.FieldBool("setuid_safe")
	d.FieldBool("root_safe")
	d.FieldBool("allow_stack_execution")
	d.FieldBool("binds_to_weak")

	d.FieldBool("weak_defines")
	d.FieldBool("canonical")
	d.FieldBool("subsections_via_symbols")
	d.FieldBool("allmodsbound")

	d.FieldBool("prebindable")
	d.FieldBool("nofixprebinding")
	d.FieldBool("nomultidefs")
	d.FieldBool("force_flat")

	d.FieldBool("twolevel")
	d.FieldBool("lazy_init")
	d.FieldBool("split_segs")
	d.FieldBool("prebound")

	d.FieldBool("bindatload")
	d.FieldBool("dyldlink")
	d.FieldBool("incrlink")
	d.FieldBool("noundefs")
}

func parseSegmentFlags(d *decode.D) {
	d.FieldRawLen("reserved", 28)
	d.FieldBool("protected_version_1")
	d.FieldBool("noreloc")
	d.FieldBool("fvmlib")
	d.FieldBool("highvm")
}

func parseSectionFlags(d *decode.D) {
	d.FieldBool("attr_pure_instructions")
	d.FieldBool("attr_no_toc")
	d.FieldBool("attr_strip_static_syms")
	d.FieldBool("attr_no_dead_strip")

	d.FieldBool("attr_live_support")
	d.FieldBool("attr_self_modifying_code")
	d.FieldBool("attr_debug")
	d.FieldRawLen("reserved", 14)

	d.FieldBool("attr_some_instructions")
	d.FieldBool("attr_ext_reloc")
	d.FieldBool("attr_loc_reloc")
}

var timestampMapper = scalar.Fn(func(s scalar.S) (scalar.S, error) {
	ts, ok := s.Actual.(uint64)
	if !ok {
		return s, nil
	}
	s.Sym = time.UnixMilli(int64(ts)).UTC().String()
	return s, nil
})

func threadStateI386Decode(d *decode.D) {
	d.FieldU32("eax")
	d.FieldU32("ebx")
	d.FieldU32("ecx")
	d.FieldU32("edx")
	d.FieldU32("edi")
	d.FieldU32("esi")
	d.FieldU32("ebp")
	d.FieldU32("esp")
	d.FieldU32("ss")
	d.FieldU32("eflags")
	d.FieldU32("eip")
	d.FieldU32("cs")
	d.FieldU32("ds")
	d.FieldU32("es")
	d.FieldU32("fs")
	d.FieldU32("gs")
}

func threadStateX8664Decode(d *decode.D) {
	d.FieldU64("rax")
	d.FieldU64("rbx")
	d.FieldU64("rcx")
	d.FieldU64("rdx")
	d.FieldU64("rdi")
	d.FieldU64("rsi")
	d.FieldU64("rbp")
	d.FieldU64("rsp")
	d.FieldU64("r8")
	d.FieldU64("r9")
	d.FieldU64("r10")
	d.FieldU64("r11")
	d.FieldU64("r12")
	d.FieldU64("r13")
	d.FieldU64("r14")
	d.FieldU64("r15")
	d.FieldU64("rip")
	d.FieldU64("rflags")
	d.FieldU64("cs")
	d.FieldU64("fs")
	d.FieldU64("gs")
}

func threadStateARM32Decode(d *decode.D) {
	rIdx := 0
	d.FieldStructArrayLoop("r", "r", func() bool {
		return rIdx < 13
	}, func(d *decode.D) {
		d.FieldU32("value")
		rIdx++
	})
	d.FieldU32("sp")
	d.FieldU32("lr")
	d.FieldU32("pc")
	d.FieldU32("cpsr")
}

func threadStateARM64Decode(d *decode.D) {
	rIdx := 0
	d.FieldStructArrayLoop("r", "r", func() bool {
		return rIdx < 29
	}, func(d *decode.D) {
		d.FieldU64("value")
		rIdx++
	})
	d.FieldU64("fp")
	d.FieldU64("lr")
	d.FieldU64("sp")
	d.FieldU64("pc")
	d.FieldU32("cpsr")
	d.FieldU32("pad")
}

func threadStatePPC32Decode(d *decode.D) {
	srrIdx := 0
	d.FieldStructArrayLoop("srr", "srr", func() bool {
		return srrIdx < 2
	}, func(d *decode.D) {
		d.FieldU32("value")
		srrIdx++
	})
	rIdx := 0
	d.FieldStructArrayLoop("r", "r", func() bool {
		return rIdx < 32
	}, func(d *decode.D) {
		d.FieldU32("value")
		rIdx++
	})
	d.FieldU32("ct")
	d.FieldU32("xer")
	d.FieldU32("lr")
	d.FieldU32("ctr")
	d.FieldU32("mq")
	d.FieldU32("vrsave")
}

func threadStatePPC64Decode(d *decode.D) {
	srrIdx := 0
	d.FieldStructArrayLoop("srr", "srr", func() bool {
		return srrIdx < 2
	}, func(d *decode.D) {
		d.FieldU64("value")
		srrIdx++
	})
	rIdx := 0
	d.FieldStructArrayLoop("r", "r", func() bool {
		return rIdx < 32
	}, func(d *decode.D) {
		d.FieldU64("value")
		rIdx++
	})
	d.FieldU32("ct")
	d.FieldU64("xer")
	d.FieldU64("lr")
	d.FieldU64("ctr")
	d.FieldU32("vrsave")
}
