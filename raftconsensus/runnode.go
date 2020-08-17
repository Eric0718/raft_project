package raftconsensus

import (
	"fmt"
	"kto/server"
	"os"

	"github.com/spf13/viper"
)

func readconfig() *cfgInfo {

	vp := viper.New()

	vp.SetConfigName("nodecfg") //把json文件换成yaml文件，只需要配置文件名 (不带后缀)即可
	vp.AddConfigPath("./conf")  //添加配置文件所在的路径
	vp.SetConfigType("yaml")    //设置配置文件类型
	err := vp.ReadInConfig()
	if err != nil {
		fmt.Printf("config file error: %s\n", err)
		os.Exit(1)
	}

	cg := &cfgInfo{
		id:        vp.GetInt("id"),
		addr:      vp.GetString("addr"),
		peer:      vp.GetString("peer"),
		peers:     vp.GetStringSlice("peers"),
		join:      vp.GetBool("join"),
		waldir:    vp.GetString("waldir"),
		snapdir:   vp.GetString("snapdir"),
		raftport:  vp.GetInt64("raftport"),
		Ds:        vp.GetString("Ds"),
		Cm:        vp.GetString("Cm"),
		QTJ:       vp.GetString("qtj"),
		snapCount: vp.GetInt64("snapCount"),

		logFile:     vp.GetString("logFile"),
		logSaveDays: vp.GetInt("logSaveDays"),
		logLevel:    vp.GetInt("logLevel"),
		logSaveMode: vp.GetInt("logSaveMode"),
		logFileSize: vp.GetInt64("logFileSize"),
	}

	return cg

}

func RunNode(s *server.Server) {

	newRaftNode(readconfig(), s)
}
