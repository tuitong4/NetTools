安装opnessl1.1.1
===================

:安装目录: /usr/local/openssl

:编译命令: ./configure --prefix=/usr/local/openssl


安装python3.7
===================

``注意启用ssl``

:安装目录: /usr/local/python3

:操作目录: python3.7源文件目录下

添加环境变量
^^^^^^^^^^^^^^^

  export CPPFLAGS="-I/usr/local/openssl/include"

  export LDFLAGS="-L/usr/local/openssl/lib"

编译python3.7
^^^^^^^^^^^^^^^^^

./configure --prefix=/usr/local/python3 --with-openssl=/usr/local/openssl

make & make install


编译python3.7动态链接库
^^^^^^^^^^^^^^^^^^^^^^^^^^^^

``只编译不安装，也可以一次性安装启动动态链接的版本python``

./configure --prefix=/usr/local/python3 --with-openssl=/usr/local/openssl --enable-shared

``目标文件夹下有重名文件libpython3.7m.a 先将该文件重命名``
mv /usr/local/python3/lib/python3.7/config-3.7m-x86_64-linux-gnu/libpython3.7m.a /usr/local/python3/lib/python3.7/config-3.7m-x86_64-linux-gnu/libpython3.7m.a.old

cp libpython3.7m.a /usr/local/python3/lib/python3.7/config-3.7m-x86_64-linux-gnu/
cp libpython3.7m.so.1.0 /usr/local/python3/lib/python3.7/config-3.7m-x86_64-linux-gnu/
cp libpython3.7m.so /usr/local/python3/lib/python3.7/config-3.7m-x86_64-linux-gnu/
cp libpython3.7.so /usr/local/python3/lib/python3.7/config-3.7m-x86_64-linux-gnu/


安装uwsgi
===================

``注意使用python3编译，不能使用python2, 源代码只是用来编译python_plugin, 安装uwsgi在全局下使用pip3安装``

:安装目录: /usr/local/python3/bin

:操作目录: uwsgi源文件目录下

删除环境变量
^^^^^^^^^^^^^^^^^

 unset $CPPFLAGS

 unset $LDFLAGS

pip3安装uwsgi
^^^^^^^^^^^^^^^^^

pip3 install uwsgi

uwsgi编译python3_plugin
^^^^^^^^^^^^^^^^^^^^^^^^^

PYTHON=python3.7 uwsgi --build-plugin "plugins/python python3"

移动python3_plugin
^^^^^^^^^^^^^^^^^^^^^^^^^

将上一步生成的python3_plugin.so 迁移至/usr/lib下或者自己的程序目录下



