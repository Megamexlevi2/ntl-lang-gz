package std

import "lunex/internal/compiler"

func RegisterAll(c *compiler.Compiler) {
	interp := c.Interpreter()

	runtimeMod := RuntimeModule(interp)
	interp.RegisterModule("runtime", runtimeMod)

	ioMod := IoModule()
	fsMod := FsModule()
	httpMod := HttpModule()
	cryptoMod := CryptoModule()
	dbMod := DbModule()
	envMod := EnvModule()
	wsMod := WsModule()
	utilsMod := UtilsModule()
	jsonMod := JsonModule()
	jwtMod := JWTModule()
	mathMod := MathModule()
	datetimeMod := DatetimeModule()
	osMod := OsModule()
	regexMod := RegexModule()
	bufferMod := BufferModule()
	intsMod := IntsModule()

	native := NativeModule()
	native.ObjVal["io"] = ioMod
	native.ObjVal["fs"] = fsMod
	native.ObjVal["http"] = httpMod
	native.ObjVal["crypto"] = cryptoMod
	native.ObjVal["db"] = dbMod
	native.ObjVal["env"] = envMod
	native.ObjVal["ws"] = wsMod
	native.ObjVal["utils"] = utilsMod
	native.ObjVal["json"] = jsonMod
	native.ObjVal["jwt"] = jwtMod
	native.ObjVal["math"] = mathMod
	native.ObjVal["datetime"] = datetimeMod
	native.ObjVal["os"] = osMod
	native.ObjVal["regex"] = regexMod
	native.ObjVal["buffer"] = bufferMod
	native.ObjVal["ints"] = intsMod
	interp.RegisterModule("internal.native", native)

	interp.RegisterModule("io", ioMod)
	interp.RegisterModule("fs", fsMod)
	interp.RegisterModule("http", httpMod)
	interp.RegisterModule("crypto", cryptoMod)
	interp.RegisterModule("db", dbMod)
	interp.RegisterModule("env", envMod)
	interp.RegisterModule("ws", wsMod)
	interp.RegisterModule("utils", utilsMod)
	interp.RegisterModule("json", jsonMod)
	interp.RegisterModule("jwt", jwtMod)
	interp.RegisterModule("math", mathMod)
	interp.RegisterModule("datetime", datetimeMod)
	interp.RegisterModule("os", osMod)
	interp.RegisterModule("regex", regexMod)
	interp.RegisterModule("buffer", bufferMod)
	interp.RegisterModule("ints", intsMod)
}
