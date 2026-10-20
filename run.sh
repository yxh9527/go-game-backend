#!/bin/bash
source base.sh

# 脚本所在目录（部署根目录）
work_pwd=$(cd "$(dirname "$0")" && pwd)

# 启动当前部署目录下的程序，并写入 pid 文件供 kill.sh 精确停止
function start(){
    if [ -n "$project_name" ] && [ "$project_name" != "all" ]; then
        game_dir=(${project_name//,/ })
    fi
    for value in ${game_dir[@]}; do
        app_dir="$work_pwd/$value"
        bin_name="${app_header}${value}"
        bin_path="$app_dir/$bin_name"
        pid_file="$app_dir/${value}.pid"

        if [ ! -x "$bin_path" ]; then
            if [ -f "$bin_path" ]; then
                chmod +x "$bin_path"
            else
                echo "skip $value: binary not found at $bin_path"
                continue
            fi
        fi

        # 启动前先停掉本目录旧进程，避免残留
        if [ -f "$pid_file" ]; then
            old_pid=$(cat "$pid_file" 2>/dev/null | tr -d ' \n\r')
            if [ -n "$old_pid" ] && kill -0 "$old_pid" 2>/dev/null; then
                exe=$(readlink -f "/proc/$old_pid/exe" 2>/dev/null || true)
                cwd=$(readlink -f "/proc/$old_pid/cwd" 2>/dev/null || true)
                if [ "$exe" = "$bin_path" ] || [ "$cwd" = "$app_dir" ]; then
                    kill -9 "$old_pid" 2>/dev/null || true
                    echo "stopped old $value pid=$old_pid"
                fi
            fi
            rm -f "$pid_file"
        fi

        cd "$app_dir" || continue
        nohup "$bin_path" run >/dev/null 2>&1 &
        echo $! > "$pid_file"
        echo "start $value ok pid=$(cat "$pid_file") dir=$app_dir"

        if [ "$value" == "data-center" ]; then
            sleep 5
        else
            sleep 2
        fi
    done
}

start
