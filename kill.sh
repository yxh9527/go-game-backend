#!/bin/bash
source base.sh

# 脚本所在目录（部署根目录），避免误杀其他目录下的同名进程
work_pwd=$(cd "$(dirname "$0")" && pwd)

# 仅杀掉「当前部署目录」下启动的进程
function kill_apps(){
    if [ -n "$project_name" ] && [ "$project_name" != "all" ]; then
        game_dir=(${project_name//,/ })
    fi
    for value in ${game_dir[@]}; do
        app_dir="$work_pwd/$value"
        bin_name="${app_header}${value}"
        bin_path="$app_dir/$bin_name"
        pid_file="$app_dir/${value}.pid"

        killed=0

        # 1) 优先按 pid 文件杀（run.sh 写入）
        if [ -f "$pid_file" ]; then
            pid=$(cat "$pid_file" 2>/dev/null | tr -d ' \n\r')
            if [ -n "$pid" ] && kill -0 "$pid" 2>/dev/null; then
                # 校验进程可执行文件仍属于本目录，防止 pid 复用误杀
                exe=$(readlink -f "/proc/$pid/exe" 2>/dev/null || true)
                cwd=$(readlink -f "/proc/$pid/cwd" 2>/dev/null || true)
                if [ "$exe" = "$bin_path" ] || [ "$cwd" = "$app_dir" ]; then
                    kill -9 "$pid" 2>/dev/null || true
                    echo "killed $value pid=$pid (from pidfile)"
                    killed=1
                else
                    echo "skip pidfile $value pid=$pid (exe/cwd mismatch: exe=$exe cwd=$cwd)"
                fi
            fi
            rm -f "$pid_file"
        fi

        # 2) 兜底：只匹配可执行路径或工作目录在本部署目录下的进程
        if [ -d /proc ]; then
            for pid in $(ps -eo pid=); do
                exe=$(readlink -f "/proc/$pid/exe" 2>/dev/null || true)
                cwd=$(readlink -f "/proc/$pid/cwd" 2>/dev/null || true)
                cmdline=$(tr '\0' ' ' < "/proc/$pid/cmdline" 2>/dev/null || true)
                # 可执行文件就是本目录二进制，或 cwd 在本服务目录且命令行包含本二进制名
                if [ "$exe" = "$bin_path" ] || \
                   { [ "$cwd" = "$app_dir" ] && echo "$cmdline" | grep -qE "(^|[ /])${bin_name}( |$)"; }; then
                    kill -9 "$pid" 2>/dev/null || true
                    echo "killed $value pid=$pid (matched local deploy)"
                    killed=1
                fi
            done
        else
            # 无 /proc 时（极少）：用绝对路径匹配，并排除 grep 自身
            pids=$(ps -ef | grep -F "$bin_path" | grep -v grep | awk '{print $2}')
            for pid in $pids; do
                kill -9 "$pid" 2>/dev/null || true
                echo "killed $value pid=$pid (path match)"
                killed=1
            done
        fi

        if [ "$killed" -eq 0 ]; then
            echo "$value not running in $app_dir"
        fi
    done
}

kill_apps
