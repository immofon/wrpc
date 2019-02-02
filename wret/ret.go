package wret

import "github.com/webasis/wrpc"

func OK(rets ...string) wrpc.Resp {
	return wrpc.Ret(wrpc.StatusOK, rets...)
}

func Error(rets ...string) wrpc.Resp {
	return wrpc.Ret(wrpc.StatusError, rets...)
}

func Auth(rets ...string) wrpc.Resp {
	return wrpc.Ret(wrpc.StatusAuth, rets...)
}

func Ban(rets ...string) wrpc.Resp {
	return wrpc.Ret(wrpc.StatusBan, rets...)
}

func IError(rets ...string) wrpc.Resp {
	return wrpc.Ret(wrpc.StatusInternalServerError, rets...)
}
