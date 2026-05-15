package clipboard

import atottoclipboard "github.com/atotto/clipboard"

type Copier interface {
	Write(text string) error
}

type SystemCopier struct{}

func (SystemCopier) Write(text string) error {
	return atottoclipboard.WriteAll(text)
}

type NoopCopier struct{}

func (NoopCopier) Write(string) error {
	return nil
}
