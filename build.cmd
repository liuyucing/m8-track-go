@echo off
REM ============================================================
REM M8 物流轨迹同步服务 一键构建脚本
REM 步骤: 1) 构建前端(Vue/Vite -> frontend\dist)  2) 编译 Go 主程序 -> bin\m8-track-go.exe
REM 用法: 双击运行, 或在项目根目录执行 build.cmd
REM       带参数 -nopause 可跳过结束暂停（供脚本/CI 调用）
REM ============================================================
setlocal
cd /d "%~dp0"

echo [1/2] 构建前端...
pushd frontend
call npm run build
if errorlevel 1 (
    echo.
    echo [错误] 前端构建失败, 请检查 Node/npm 环境
    popd
    exit /b 1
)
popd

echo.
echo [2/2] 编译 Go 主程序...
go build -ldflags="-s -w -H windowsgui" -o bin\m8-track-go.exe .
if errorlevel 1 (
    echo.
    echo [错误] Go 编译失败 ^(若提示文件被占用, 请先关闭正在运行的 m8-track-go.exe^)
    exit /b 1
)

echo.
echo [完成] 构建成功: bin\m8-track-go.exe
REM 双击运行时停一下看结果
if /i not "%~1"=="-nopause" pause
endlocal
