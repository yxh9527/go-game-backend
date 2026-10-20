#!/bin/bash
export GOOS=linux
source base.sh
#获取当前sh目录
work_pwd=`pwd`
#编译所有
function build(){    
    #遍历更新
    for value in ${game_dir[@]}; do 
    {
        gd=$work_pwd"/"$value
        cd $gd
        if [ $value == "hall" ]; then
            cd "cmd"&&cd "hall"
        fi
        echo `pwd`
        go build -ldflags="-w -s" -tags $platform
    }
    done
    wait
    echo "编译完成 等待打包..."
}

# 打包
function package(){
    rm -rf $work_pwd"/"release-*
    p=$work_pwd"/__yxh2023_pakage_tmp"
    rm -rf $p && mkdir $p
    datestr="release-"`date +"%Y%m%d%H%M"`
    for value in ${game_dir[@]}; do
        if [ $value == "auth" ]; then
            mkdir $p"/auth" && mv $work_pwd/"auth/cmd/auth/auth" $p"/auth/"$app_header"auth"&& chmod +x *
        elif [ $value == "hall" ]; then
            mkdir $p"/hall" && mv $work_pwd/"hall/cmd/hall/hall" $p"/hall/"$app_header"hall"&& chmod +x *
        else 
            mkdir $p"/"$value && mv $work_pwd"/"$value"/"$value $p"/"$value"/"$app_header$value && chmod +x *
        fi
        cp $work_pwd"/"$value"/config.yaml" $p"/"$value
         if [ -d $work_pwd"/"$value"/data" ];then
            cp -rf $work_pwd"/"$value"/data" $p"/"$value
        fi
    done
    cp $work_pwd/base.sh $p
    cp $work_pwd/kill.sh $p
    cp $work_pwd/run.sh  $p

    chmod +x $p/*.sh

    mv $p $work_pwd"/"$datestr && cd $work_pwd  && tar -zcvf $datestr.tgz $datestr
    rm -rf $work_pwd"/"$datestr
    
    echo "打包完成"
}

# 编译
build
#打包
package
