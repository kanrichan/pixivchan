# PixivChan


### 🚀Support
- `github.com`
- `pixiv.net`

### 📝使用步骤
#### Chrome
1. 启动程序（自动生成根证书颁发机构 `ca.cer` ）
2. 添加PAC代理设置
   - 关闭所有 `Chrome` 窗口（否则无法应用设置）
   - 找到 `Chrome` 程序并新建 `Chrome` 快捷方式
   - 右键打开 `属性`
   - 在 `目标(T)` 末尾插入 `--proxy-pac-url=http://127.0.0.1:8080/pixivchan.pac`
3. 信任根证书颁发机构 `ca.cer`
   - 打开链接 `chrome://settings/security`
   - 找到 `管理设备证书`
   - 选择 `受信任的根证书颁发机构`
   - 点击 `导入`
   - 选择项目根目录下的 `ca.cer` 
4. 打开浏览器，访问网站，如 `https://github.com` 、 `https://pixiv.net` 

#### FireFox
1. 启动程序（自动生成根证书颁发机构 `ca.cer` ）
2. 添加PAC代理设置
   - 打开链接 `about:preferences#general`
   - 找到 `网络设置` ，点击 `设置`
   - 选择 `自动代理配置的 URL (PAC)`
   - 在输入框内输入 `http://127.0.0.1:8080/pixivchan.pac`
   - 点击 `重新载入` ，点击 `确定`
3. 信任根证书颁发机构 `ca.cer`
   - 打开链接 `about:preferences#privacy`
   - 找到 `证书` ，点击 `查看证书`
   - 选择 `证书颁发机构`
   - 点击 `导入`
   - 选择项目根目录下的 `ca.cer`
   - 点击 `确定`
4. 再次启动程序，浏览器访问网站，如 `https://github.com` 、 `https://pixiv.net` 
