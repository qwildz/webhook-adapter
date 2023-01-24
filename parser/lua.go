package parser

import (
	"context"
	"log"
	"net/http"
	"runtime"

	"github.com/cjoudrey/gluahttp"
	"github.com/jackc/puddle/v2"
	lua "github.com/yuin/gopher-lua"
	luajson "layeh.com/gopher-json"
)

var luaPool = newLuaParser()
var parsedLua = make(map[uint]*lua.LFunction)

func newLuaParser() *puddle.Pool[*lua.LState] {
	maxPoolSize := int32(runtime.NumCPU())

	constructor := func(context.Context) (*lua.LState, error) {
		luaState := lua.NewState()
		luaState.PreloadModule("http", gluahttp.NewHttpModule(&http.Client{}).Loader)
		luajson.Preload(luaState)
		return luaState, nil
	}

	destructor := func(value *lua.LState) {
		value.Close()
	}

	pool, err := puddle.NewPool(&puddle.Config[*lua.LState]{
		Constructor: constructor,
		Destructor:  destructor,
		MaxSize:     maxPoolSize,
	})

	if err != nil {
		log.Fatalln(err)
	}

	return pool
}

func RunLua(ctx context.Context, id uint, script, payload string) (string, error) {
	res, _ := luaPool.Acquire(ctx)
	defer res.Release()

	var luaInstance = res.Value()

	_, exists := parsedLua[id]

	if !exists {
		fn, err := luaInstance.LoadString(script)

		if err != nil {
			log.Fatalln(err)
		}

		parsedLua[id] = fn
	}

	luaInstance.Push(parsedLua[id])

	if err := luaInstance.PCall(0, lua.MultRet, nil); err != nil {
		log.Fatalln(err)
	}

	luaInstance.SetGlobal("payload", lua.LString(payload))
	if err := luaInstance.DoString(`return require("json").encode(transform(payload))`); err != nil {
		log.Fatalln(err)
	}

	ret := luaInstance.Get(-1)
	luaInstance.Pop(1)

	return ret.String(), nil
}
