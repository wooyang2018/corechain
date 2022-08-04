package logger

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/wooyang2018/corechain/logger/mlog"
)

// OpenMLog create and open log stream using LogConfig
func OpenMLog(lc *LogConf, logDir string) (LogDriver, error) {
	infoFile := filepath.Join(logDir, lc.Filename+".log")
	wfFile := filepath.Join(logDir, lc.Filename+".log.wf")
	os.MkdirAll(logDir, os.ModePerm)

	var lfmt mlog.Format
	switch lc.Fmt {
	case "json":
		lfmt = mlog.JsonFormat()
	case "logfmt":
		lfmt = mlog.LogfmtFormat()
	}

	xlog := mlog.New("module", lc.Module)
	lvLevel, err := mlog.LvlFromString(lc.Level)
	if err != nil {
		return nil, fmt.Errorf("log level error.err:%v", err)
	}
	// set lowest level as level limit, this may improve performance
	xlog.SetLevelLimit(lvLevel)

	// init normal and warn/fault log file handler, RotateFileHandler
	// only valid if `RotateInterval` and `RotateBackups` greater than 0
	var nmHandler, wfHandler mlog.Handler
	if lc.RotateInterval > 0 && lc.RotateBackups > 0 {
		nmHandler = mustBufferFileHandler(
			infoFile, lfmt, lc.RotateInterval, lc.RotateBackups)
		wfHandler = mustBufferFileHandler(
			wfFile, lfmt, lc.RotateInterval, lc.RotateBackups)
	} else {
		nmHandler = mlog.Must.FileHandler(infoFile, lfmt)
		wfHandler = mlog.Must.FileHandler(wfFile, lfmt)
	}

	if lc.Async {
		nmHandler = mlog.BufferedHandler(lc.BufSize, nmHandler)
		wfHandler = mlog.BufferedHandler(lc.BufSize, wfHandler)
	}

	// prints log level between `lvLevel` to Info to base log
	nmfileh := mlog.LvlFilterHandler(lvLevel, nmHandler)
	// prints log level greater or equal to Warn to wf log
	wffileh := mlog.LvlFilterHandler(mlog.LvlWarn, wfHandler)

	var lhd mlog.Handler
	if lc.Console {
		hstd := mlog.StreamHandler(os.Stderr, lfmt)
		lhd = mlog.MultiHandler(hstd, nmfileh, wffileh)
	} else {
		lhd = mlog.MultiHandler(nmfileh, wffileh)
	}
	xlog.SetHandler(lhd)

	return xlog, err
}
