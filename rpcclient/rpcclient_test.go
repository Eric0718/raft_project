package rpcclient

import (
	"context"
	"fmt"
	"kto/rpcclient/message"
	"kto/transaction"
	"kto/types"
	"kto/until"
	"testing"
	"time"

	"google.golang.org/grpc"
)

var ctx context.Context
var client message.GreeterClient

func init() {
	from := types.BytesToAddress([]byte("CNtyeiE8RkTy26ufueMnvvbkEJ5qQL7tjD8Su5BP8PLY"))
	to := types.BytesToAddress([]byte("HRBfv2SA5BBiweBShPMbDYFfr6kKwyk9c9pxpD481fMc"))
	fromPri := until.Decode("ziNi4JJCU29sWRFUVyeTZRkhbZetNd39wGaEbMikGTg2y3EjASoafN1ArnTMqk1AovGTVUT8o8RdBRPSTNa162Y")
	tx := transaction.New()
	tx = tx.NewTransaction(uint64(1), 200000, from, to, "")
	tx = tx.SignTx(fromPri)
	_, err := tx.SendTransaction()
	if err != nil {
		fmt.Println(err)
	}

	conn, err := grpc.Dial("106.13.188.227:8545", grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		fmt.Println("fail to dail:", err)
		return
	}
	client = message.NewGreeterClient(conn)
	ctx = context.Background()
}

func TestSetLockBalance(t *testing.T) {
	time.Sleep(5 * time.Second)
	var reqData = &message.ReqLockBalance{Address: "CNtyeiE8RkTy26ufueMnvvbkEJ5qQL7tjD8Su5BP8PLY",
		Amount: uint64(10000)}
	respdata, err := client.SetLockBalance(ctx, reqData)
	if err != nil {
		t.Error("fail to set lcok:", err)
	}
	t.Log(respdata.Status)
}

func TestSetUnlockBalance(t *testing.T) {
	var reqData = &message.ReqUnlockBalance{Address: "CNtyeiE8RkTy26ufueMnvvbkEJ5qQL7tjD8Su5BP8PLY",
		Amount: uint64(5000)}
	respdata, err := client.SetUnlockBalance(ctx, reqData)
	if err != nil {
		t.Error("fail to set unlock:", err)
	}
	t.Log(respdata.Status)
}

func TestGetBalance(t *testing.T) {
	_, err := client.GetBalance(context.Background(), &message.ReqBalance{Address: "Kto3fwLDibEp5Ggx51A9fJ2h5KAJvargAA31JWae7D6PPbj"})
	if err != nil {
		t.Errorf("failed,error:%v", err)
		return
	}
	//t.Log("balance:", respdata.Balnce)
}
