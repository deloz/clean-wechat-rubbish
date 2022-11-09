package main

import (
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
)

var (
	wg sync.WaitGroup
	f  = afero.NewOsFs()
	ch = make(chan string)
)

func main() {
	initLog()

	quitChan := make(chan os.Signal, 1)
	signal.Notify(quitChan, syscall.SIGINT, syscall.SIGTERM)

	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal().
			AnErr("err", err).
			Msg("get user home directory failed")
		return
	}

	wechatRootDir := filepath.Join(homeDir, "Documents", "WeChat Files")

	go func() {
		for i := range ch {
			if err := removeContents(i); err != nil {
				log.Error().
					AnErr("err", err).
					Msg("delete failed")
			} else {
				log.Info().
					Msgf("OK: %s", i)
			}
		}

		log.Info().
			Msg("all done!")
	}()

	if err := removeDirectories(wechatRootDir); err != nil {
		log.Error().
			AnErr("err", err).
			Msg("remove failed")
		return
	}

	wg.Wait()

	close(ch)

	<-quitChan
}

func removeContents(dir string) error {
	d, err := f.Open(dir)
	if err != nil {
		return err
	}

	defer func() {
		if err := d.Close(); err != nil {
			log.Error().
				AnErr("err", err).
				Msg("close failed")
		}
	}()

	names, err := d.Readdirnames(-1)
	if err != nil {
		return err
	}

	for _, name := range names {
		if err = f.RemoveAll(filepath.Join(dir, name)); err != nil {
			log.Error().
				AnErr("err", err).
				Msg("remove all failed")
		}
	}

	return nil
}

func removeDirectories(wechatRootDir string) error {

	infos, err := afero.ReadDir(f, wechatRootDir)
	if err != nil {
		log.Error().
			AnErr("err", err).
			Msg("read wechat files directories failed")
		return err
	}

	subDirNames := []string{"Image", "Video", "Temp", "MsgAttach", "File", "CustomEmotion", "Cache", "Sns"}

	for _, info := range infos {
		if info.IsDir() {
			for _, s := range subDirNames {
				wg.Add(1)

				dir := filepath.Join(wechatRootDir, info.Name(), "FileStorage", s)

				go foo(ch, dir)
			}
		}
	}

	return nil
}

func foo(c chan string, dir string) {
	defer wg.Done()

	c <- dir
}

func initLog() {
	zerolog.CallerMarshalFunc = func(pc uintptr, file string, line int) string {
		short := file
		for i := len(file) - 1; i > 0; i-- {
			if file[i] == '/' {
				short = file[i+1:]
				break
			}
		}
		file = short
		return file + ":" + strconv.Itoa(line)
	}
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: "2006-01-02 15:04:05"}).With().Caller().Logger()
}
