gomod设置
--------

设置GOPROXY代理：
go env -w GOPROXY=https://goproxy.cn,direct
如果报错：执行unset GOPROXY


禁用SUM：
go env -w GOSUMDB=off
