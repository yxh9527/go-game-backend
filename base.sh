#!/bin/bash
#export GOOS=linux
#定义项目名称,
#fk
game_dir_go="data-center,web-api,open-api,lottery,game-data-summary,client-api"

#定义游戏项目前缀方便后面关闭脚本使用
#app_header=__yxh_
app_header=
#第一个参数指定平台 236 lg ly
platform=$1
#第二个参数指定单个编译时项目名称以“,”分割
project_name=$2
game_dir={}
#判断是否制定了打包项目 项目名称以“,”分割
if [ -n "$project_name" ] && [ "$project_name" != "all" ]; then
    game_dir=(${project_name//,/ })
else
    tmp="game_dir_"$platform
    app=$(eval echo \$$tmp)
    game_dir=(${app//,/ })
fi
