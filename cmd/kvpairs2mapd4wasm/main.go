package main

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"strconv"

	pm "github.com/takanoriyanagitani/go-cbor-kvpairs2mapd4wasm"
	rf "github.com/takanoriyanagitani/go-cbor-kvpairs2mapd4wasm/cbor/dec/rdr2kvpairs/fa"
	pf "github.com/takanoriyanagitani/go-cbor-kvpairs2mapd4wasm/cbor/enc/pairs2wtr/fa"
	w2 "github.com/takanoriyanagitani/go-cbor-kvpairs2mapd4wasm/mapper/wasm/wazero"
	"github.com/tetratelabs/wazero"
)

var rdr io.Reader = bufio.NewReader(os.Stdin)

var wasmPath string = os.Getenv("ENV_WASM_PATH")

var swasmSizMax string = os.Getenv("ENV_WASM_SIZE_MAX")

var mapperName string = os.Getenv("ENV_WASM_MAPPER_NAME")

func path2bytes(limit int64) func(string) ([]byte, error) {
	return func(filepat string) ([]byte, error) {
		file, err := os.Open(filepat)
		if nil != err {
			return nil, err
		}
		defer file.Close()

		lmtd := &io.LimitedReader{R: file, N: limit}
		var buf bytes.Buffer
		_, err = io.Copy(&buf, lmtd)
		return buf.Bytes(), err
	}
}

//nolint:funlen
func sub(ctx context.Context) error {
	slog.Info("setting up", "ENV_WASM_PATH", wasmPath)
	slog.Info("setting up", "ENV_WASM_SIZE_MAX", swasmSizMax)
	slog.Info("setting up", "ENV_WASM_MAPPER_NAME", mapperName)

	iwasmSizMax, err := strconv.Atoi(swasmSizMax)
	if nil != err {
		return err
	}

	iwsm := int64(iwasmSizMax)
	slog.Info("setting up", "wasm-size-max", int64(iwasmSizMax))

	var fpat2bytes func(string) ([]byte, error) = path2bytes(iwsm)
	wasmBytes, err := fpat2bytes(wasmPath)
	if nil != err {
		return err
	}
	slog.Info("setting up", "wasm-size", len(wasmBytes))

	var wrtm wazero.Runtime = wazero.NewRuntime(ctx)
	defer wrtm.Close(ctx)
	var wcfg wazero.ModuleConfig = wazero.NewModuleConfig()

	var whmb wazero.HostModuleBuilder = wrtm.NewHostModuleBuilder("host")
	var emptyState w2.State
	whmb = emptyState.RegisterAll(whmb)
	hmod, err := whmb.Instantiate(ctx)
	if nil != err {
		return err
	}

	wbytes := w2.WasmBytes(wasmBytes)
	compiled, err := wbytes.ToCompiled(ctx, wrtm)
	if nil != err {
		return err
	}

	slog.Info("module compiled", "name", compiled.CompiledModule.Name())

	slog.Info("host module instantiated", "name", hmod.Name())

	wmod, err := compiled.ToModule(ctx, wrtm, wcfg)
	if nil != err {
		return err
	}

	slog.Info("module instantiated", "name", wmod.Module.Name())

	wstate, err := wmod.ToState(mapperName)
	if nil != err {
		return err
	}

	var bwtr *bufio.Writer = bufio.NewWriter(os.Stdout)
	var r2s pm.ReaderToSource = rf.ReaderToSource
	var w2w pm.WtrToPairsWriter = pf.WtrToPairsWriter

	var pmap pm.PairMapper = emptyState.ToPairMapper()
	emptyState.Function = wstate.Function

	r2s2p2m2w := pm.ReaderToSourceToPairsToMapdToWriter{
		ReaderToSource:   r2s,
		PairMapper:       pmap,
		WtrToPairsWriter: w2w,
	}

	return errors.Join(
		r2s2p2m2w.ReaderToWriter(
			ctx,
			rdr,
			bwtr,
		),
		bwtr.Flush(),
	)
}

func main() {
	err := sub(context.Background())
	if nil != err {
		slog.Error("error got", "err", err)
	}
}
