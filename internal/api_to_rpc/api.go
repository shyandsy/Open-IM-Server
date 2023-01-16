package common

import (
	api "Open_IM/pkg/base_info"
	"Open_IM/pkg/common/config"
	"Open_IM/pkg/common/log"
	"Open_IM/pkg/common/trace_log"
	"Open_IM/pkg/getcdv3"
	rpc "Open_IM/pkg/proto/friend"
	utils2 "Open_IM/pkg/utils"
	"context"
	"fmt"
	utils "github.com/OpenIMSDK/open_utils"
	"github.com/gin-gonic/gin"
	"reflect"
)

func a(c *gin.Context) {
	ApiToRpc(c, &api.AddBlacklistReq{}, &api.AddBlacklistResp{}, config.Config.RpcRegisterName.OpenImFriendName, rpc.NewFriendClient, utils.GetSelfFuncName())
}

func ApiToRpc(c *gin.Context, apiReq, apiResp interface{}, rpcName string, rpcClientFunc interface{}, rpcFuncName string) {
	logFuncName := fmt.Sprintf("[ApiToRpc: %s]%s", utils2.GetFuncName(1), rpcFuncName)
	ctx := trace_log.NewCtx1(c, rpcFuncName)
	defer log.ShowLog(ctx)
	if err := c.BindJSON(apiReq); err != nil {
		trace_log.WriteErrorResponse(ctx, "BindJSON", err)
		return
	}
	trace_log.SetCtxInfo(ctx, logFuncName, nil, "apiReq", apiReq)
	etcdConn, err := getcdv3.GetConn(ctx, rpcName)
	if err != nil {
		trace_log.WriteErrorResponse(ctx, "GetConn", err)
		return
	}
	rpcClient := reflect.ValueOf(rpcClientFunc).Call([]reflect.Value{
		reflect.ValueOf(etcdConn),
	})[0].MethodByName(rpcFuncName) // rpcClient func
	rpcReqPtr := reflect.New(rpcClient.Type().In(1).Elem()) // *req
	CopyAny(apiReq, rpcReqPtr.Interface())
	trace_log.SetCtxInfo(ctx, logFuncName, nil, "opUserID", c.GetString("opUserID"), "callRpcReq", rpcString(rpcReqPtr.Elem().Interface()))
	respArr := rpcClient.Call([]reflect.Value{
		reflect.ValueOf(context.Context(c)), // context.Context (ctx operationID. opUserID)
		rpcReqPtr,                           // rpcClient apiReq
	}) // respArr => (apiResp, error)
	if !respArr[1].IsNil() { // rpcClient err != nil
		err := respArr[1].Interface().(error)
		trace_log.WriteErrorResponse(ctx, rpcFuncName, err, "callRpcResp", "error")
		return
	}
	rpcResp := respArr[0].Elem()
	trace_log.SetCtxInfo(ctx, rpcFuncName, nil, "callRpcResp", rpcString(rpcResp.Interface()))
	if apiResp != nil {
		CopyAny(rpcResp.Interface(), apiResp)
	}
	trace_log.SetSuccess(ctx, rpcFuncName, apiResp)
}

func rpcString(v interface{}) string {
	if s, ok := v.(interface{ String() string }); ok {
		return s.String()
	}
	return fmt.Sprintf("%+v", v)
}

//func ApiToRpc(c *gin.Context, apiReq, apiResp interface{}, rpcName string, fn interface{}, rpcFuncName string, tokenFunc func(token string, operationID string) (string, error)) {
//	nCtx := trace_log.NewCtx(c, rpcFuncName)
//	defer trace_log.ShowLog(nCtx)
//	if err := c.BindJSON(apiReq); err != nil {
//		trace_log.WriteErrorResponse(nCtx, "BindJSON", err)
//		return
//	}
//	reqValue := reflect.ValueOf(apiReq).Elem()
//	operationID := reqValue.FieldByName("OperationID").String()
//	trace_log.SetOperationID(nCtx, operationID)
//	trace_log.SetCtxInfo(nCtx, "BindJSON", nil, "params", apiReq)
//	etcdConn, err := utils2.GetConn(c, rpcName)
//	if err != nil {
//		trace_log.WriteErrorResponse(nCtx, "GetDefaultConn", err)
//		return
//	}
//	rpc := reflect.ValueOf(fn).Call([]reflect.Value{
//		reflect.ValueOf(etcdConn),
//	})[0].MethodByName(rpcFuncName) // rpc func
//	rpcReqPtr := reflect.New(rpc.Type().In(1).Elem()) // *req参数
//	var opUserID string
//	if tokenFunc != nil {
//		var err error
//		opUserID, err = tokenFunc(c.GetHeader("token"), operationID)
//		if err != nil {
//			trace_log.WriteErrorResponse(nCtx, "TokenFunc", err)
//			return
//		}
//	}
//	if opID := rpcReqPtr.Elem().FieldByName("OperationID"); opID.IsValid() {
//		opID.SetString(operationID)
//		if opU := rpcReqPtr.Elem().FieldByName("OpUserID"); opU.IsValid() {
//			opU.SetString(opUserID)
//		}
//	} else {
//		op := rpcReqPtr.Elem().FieldByName("Operation").Elem()
//		op.FieldByName("OperationID").SetString(operationID)
//		op.FieldByName("OpUserID").SetString(opUserID)
//	}
//	if err := utils.CopyStructFields(rpcReqPtr.Interface(), apiReq); err != nil {
//		trace_log.WriteErrorResponse(nCtx, "CopyStructFields_RpcReq", err)
//		return
//	}
//	respArr := rpc.Call([]reflect.Value{
//		reflect.ValueOf(context.Context(c)), // context.Context
//		rpcReqPtr,                           // rpc apiReq
//	}) // respArr => (apiResp, error)
//	if !respArr[1].IsNil() { // rpc err != nil
//		err := respArr[1].Interface().(error)
//		trace_log.WriteErrorResponse(nCtx, rpcFuncName, err, "rpc req", rpcReqPtr.Interface())
//		return
//	}
//	rpcResp := respArr[0].Elem()
//	trace_log.SetCtxInfo(nCtx, rpcFuncName, nil, "rpc req", rpcReqPtr.Interface(), "resp", rpcResp.Interface())
//	commonResp := rpcResp.FieldByName("CommonResp").Elem()
//	errCodeVal := commonResp.FieldByName("ErrCode")
//	errMsgVal := commonResp.FieldByName("ErrMsg").Interface().(string)
//	errCode := errCodeVal.Interface().(int32)
//	if errCode != 0 {
//		trace_log.WriteErrorResponse(nCtx, "RpcErrCode", &constant.ErrInfo{
//			ErrCode: errCode,
//			ErrMsg:  errMsgVal,
//		})
//		return
//	}
//	if apiResp != nil {
//		if err := utils.CopyStructFields(apiResp, rpcResp.Interface()); err != nil {
//			trace_log.WriteErrorResponse(nCtx, "CopyStructFields_RpcResp", err)
//			return
//		}
//	}
//	trace_log.SetSuccess(nCtx, rpcFuncName, apiResp)
//}
