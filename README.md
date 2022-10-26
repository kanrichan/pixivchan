# PixivChan

### 使用步骤
1. `Host` 文件添加以下内容
   ```
   127.0.0.1 pixiv.net
   127.0.0.1 www.pixiv.net
   127.0.0.1 imp.pixiv.net
   127.0.0.1 accounts.pixiv.net
   127.0.0.1 i.pximg.net
   127.0.0.1 s.pximg.net
   127.0.0.1 a.pixiv.org
   127.0.0.1 github.com
   127.0.0.1 api.github.com
   127.0.0.1 collector.github.com
   127.0.0.1 github.githubusercontent.com
   127.0.0.1 avatars.githubusercontent.com
   127.0.0.1 raw.githubusercontent.com
   127.0.0.1 github.githubassets.com
   127.0.0.1 repository-images.githubusercontent.com
   ```
2. 启动一次程序并关闭（生成证书）
3. 信任 `ca.cer` 为 `受信任的根证书颁发机构`
4. 再次启动程序，浏览器访问网站