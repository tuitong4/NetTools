1、选择合适的镜像
jupyter/base-notebook
jupyter/minimal-notebook
jupyter/r-notebook
jupyter/scipy-notebook
jupyter/tensorflow-notebook
jupyter/datascience-notebook
jupyter/pyspark-notebook
jupyter/all-spark-notebook


2、获取密码, 拷贝输出内容
``` python
$ from notebook.auth import passwd

$ passwd()


```

3、启用容器
``` shell
docker run -d -p 8888:8888 -v /data/jupyter:/home/jovyan/work jupyter/base-notebook start-notebook.sh --NotebookApp.password='argon2:$argon2id$v=19$m=10240,t=10,p=8$6tumQ0onKk/xr47JpMD2pQ$+TRgGQrVTE9C5kZL42uq6A'
```
