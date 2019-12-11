package feednotifier

import "testing"

func TestFileReader(t *testing.T) {
	count := 0
	err := ReadLines("test/third.xml", " \r\n", func(l string) error {
		count++
		return nil
	})
	if err != nil {
		t.Fail()
	}
	if count != 59 {
		t.Fail()
	}
}
func TestFileReaderInvalidFile(t *testing.T) {
	err := ReadLines("doesnotexist", " \r\n", func(l string) error {
		t.Fail()
		return nil
	})
	if err == nil {
		t.Fail()
	}
}
func TestFileReaderEmptyFile(t *testing.T) {
	count := 0
	err := ReadLines("test/empty", " \r\n", func(l string) error {
		count++
		t.Fail()
		return nil
	})
	if err != nil {
		t.Fail()
	}
	if count > 0 {
		t.Fail()
	}
}
func TestFileReaderNonBlankLastLine(t *testing.T) {
	count := 0
	err := ReadLines("test/filewithnonblanklastline", " \r\n", func(l string) error {
		count++
		return nil
	})
	if err != nil {
		t.Fail()
	}
	if count != 4 {
		t.Fail()
	}
}
