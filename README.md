

1. 安装 oracle instant client

	将 instantclient_12_1.tar 上传到 /opt 目录下
	tar -xvpf instantclient_12_1.tar
	进入 /opt/instantclient_12_1 修改文件 tnsnames.ora 添加条目

2. 环境变量
	切换当前目录到用户home 目录
	cd ~
	vi .bash_profile 或者在执行程序前设置下面变量

	export LD_LIBRARY_PATH=/opt/instantclient_12_1

	export TNS_ADMIN=/opt/instantclient_12_1

	source .bash_profile

3. 修改load.conf
	
	3.1 修改 uid 配置以确保能连接到查验用户
	3.2 修改 proxy 区段,每行表示一个月份的代理 url