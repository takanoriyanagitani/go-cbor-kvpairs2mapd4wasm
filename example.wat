(module

	(import "host" "key2wasm" (func $key2wasm (param $keyptr i32) (result i32)))
	(import "host" "val2wasm" (func $val2wasm (param $valptr i32) (result i32)))

	(import "host" "wasm2key" (func $wasm2key (param i32 i32)(result i32)))
	(import "host" "wasm2val" (func $wasm2val (param i32 i32)(result i32)))

	;; (import "host" "wasm2key" (func $wasm2key (param $ptr i32) (param $siz i32)))
	;; (import "host" "wasm2val" (func $wasm2val (param $ptr i32) (param $siz i32)))

	(memory (export "memory") 4)

	(global $IKEY_PTR i32 (i32.const 0x0000_0000))
	(global $IVAL_PTR i32 (i32.const 0x0001_0000))

	(global $OKEY_PTR i32 (i32.const 0x0002_0000))
	(global $OVAL_PTR i32 (i32.const 0x0003_0000))

	(global $gcnt (mut i32) (i32.const 0))

	(func $map_kvpair (export "map_kvpair")
	  (local $keylen i32)
	  (local $vallen i32)

	  (local $skret i32)
	  (local $svret i32)

		global.get $IKEY_PTR call $key2wasm local.set $keylen
		global.get $IVAL_PTR call $val2wasm local.set $vallen

		local.get $keylen i32.const 0 i32.lt_s if unreachable end
		local.get $vallen i32.const 0 i32.lt_s if unreachable end

		global.get $OKEY_PTR global.get $IKEY_PTR local.get $keylen memory.copy
		global.get $OVAL_PTR global.get $IVAL_PTR local.get $vallen memory.copy

		local.get $vallen global.get $gcnt i32.sub local.set $vallen

		global.get $OKEY_PTR local.get $keylen call $wasm2key local.set $skret
		global.get $OVAL_PTR local.get $vallen call $wasm2val local.set $svret

		local.get $skret i32.const 0 i32.ne if unreachable end
		local.get $svret i32.const 0 i32.ne if unreachable end

		global.get $gcnt i32.const 1 i32.add global.set $gcnt
  )

)
