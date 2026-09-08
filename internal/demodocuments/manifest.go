// Package demodocuments identifies immutable fictional files used by the demo.
// A manifest match is not an antivirus result or an evidence review decision.
package demodocuments

import "embed"

//go:embed assets/*.pdf assets/*.png assets/*.xlsx
var assets embed.FS

type File struct {
	Name, MediaType, SHA256 string
	SizeBytes               int64
}

var manifest = [...]File{
	{"sample-insurance-schedule.pdf", "application/pdf", "14520f835f3edb3a14c6e515ff1c470e93474f9d7641ccdef8a458f04571b2d4", 2414},
	{"sample-office-statement.png", "image/png", "0b83d191c1d01b0cc03fea396dcee9a260611f18395cc464089c1959eb1f8883", 111799},
	{"sample-recovery-plan.pdf", "application/pdf", "d42fdd1a235d1b366f6ea39e5bad062f432518c4a1ac6bad3c97e84da918945a", 2700},
	{"sample-security-declaration-previous.pdf", "application/pdf", "9dc70c813e11549d0c73f316fada2fa2249d3d521e0e7945d37c2cc4be9d04c9", 2529},
	{"sample-security-declaration.pdf", "application/pdf", "828c7e74c9ec872169a2e477ce0282ac9de8578d62d470c5f8535516bed570f8", 2629},
	{"sample-subprocessor-register.xlsx", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", "a8d02f4ecb42893fdda5ca8a75a248dc85924be50f494485b16f19ff21ec16b7", 4461},
}

func Files() []File { return append([]File(nil), manifest[:]...) }

func Read(name string) ([]byte, bool) {
	for _, file := range manifest {
		if file.Name == name {
			content, err := assets.ReadFile("assets/" + name)
			return content, err == nil
		}
	}
	return nil, false
}

func Matches(name, mediaType, digest string, size int64) bool {
	for _, file := range manifest {
		if file.Name == name && file.MediaType == mediaType && file.SHA256 == digest && file.SizeBytes == size {
			return true
		}
	}
	return false
}
