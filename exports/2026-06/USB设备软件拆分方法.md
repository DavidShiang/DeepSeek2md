# USB设备软件拆分方法

### 🧑 用户

一个usb设备，是否可以通过软件让系统认为是2个，也就是在硬件层之上应用驱动之下达成这个效果。

---

### 🤖 DeepSeek

可以的。这种效果通常称为 **USB 设备复合（Composite Device）** 或 **多功能设备拆分**，是通过软件在 **驱动层** 或 **固件/中间层** 实现的，不需要改变物理硬件。

具体实现方式取决于你的场景：

### 1. 最常见的方案：USB 重定向 / 虚拟拆分（应用+驱动层）
用一个软件将物理 USB 设备的各个功能（如键盘+存储、多个串口）强行拆成两个独立的“虚拟 USB 设备”供系统调用。

- **Windows 范例：**  
  - **USB over Ethernet 类软件（如 USB Network Gate）**：可以共享一个 USB 设备的某一功能，但不是直接本地拆成两个。  
  - **虚拟串口拆分器（如 Virtual Serial Port Emulator, com0com）**：把一个物理 USB 转串口设备变成两个虚拟 COM 口，分别对应用层可见。  
  - **HID 设备拆分（如 HidHide）**：原本用于游戏手柄遮挡，但原理上能把一个 HID 设备在系统设备列表中隐藏，并通过虚拟驱动生成多个副本。
- **效果：** 系统上会看到两个独立的设备实例（例如“USB 输入设备 #1”和“#2”），但它们共享同一物理信号。

### 2. 底层方案：修改 USB 设备固件（硬件层之上，驱动之下）
如果设备本身是微控制器（如 Arduino、STM32、Cypress），你可以在固件中把它配置为 **USB 复合设备** 并且报告两个不同的接口（Interface）或两个逻辑单元（如 MSC + CDC 或 2×CDC）。此时：

- 插入电脑后，系统原生就会识别为两个独立的功能设备。
- **不需要额外安装上层驱动**，这是最“正统”的软件/固件实现方式。

### 3. 中间层驱动过滤（Filter Driver）
写一个 **Windows 内核过滤驱动** 或 **Linux USB gadget 子系统/ usbip 改装**：

- 截获物理 USB 设备的请求，为系统创建第二个 PDO（物理设备对象）。
- 例如使用 **libusb / WinUSB** 配合虚拟总线驱动（如 **Virtual Bus Driver**）可以模拟一个额外的设备节点。

### 需要注意的限制（为什么并不普遍）

- **物理带宽共享**：两个“虚拟”设备共用同一个 USB 总线的端点（Endpoint），如果同时高负载传输，可能会互相影响。
- **独立 Reset 失败**：系统无法单独 Reset 其中一个虚拟设备而不影响另一个。
- **操作系统限制**：系统设备树中一个物理 USB 设备通常对应一个父节点。要完全变成两个“看起来像插了不同端口”的设备，需要写 **虚拟 USB 主控制器驱动**（非常复杂，且需签名），一般只有虚拟机软件（如 VirtualBox USB passthrough + filter）能做到。

### 推荐方案总结（从易到难）

| 目标 | 推荐方法 |
|------|-----------|
| 让普通应用看到两个串口、两个 HID 等 | **虚拟端口软件**（com0com, HidHide, VSPE） |
| 自己做的嵌入式 USB 设备 | **修改固件为复合设备**（最简单且原生） |
| 通用任意现有 USB 设备（不改固件）在本地拆成两个逻辑设备 | **开发一个 Filter Driver**，或者用 **USB/IP + 自定义后端**（复杂） |
| 完全隔离：两个独立的设备实例（甚至不同驱动） | **硬件级**：实际用 USB HUB + 两个物理设备，或 **PCIe 重映射 + SR-IOV-like USB**（不存在通用方案） |

**结论：**  
可以通过软件/驱动做到，但最常见且实用的是 **方案 1（虚拟端口软件）** 或 **方案 2（固件复合设备）**。如果你有一个具体设备型号和使用场景，我可以给出更精确的配置建议。

---

### 🧑 用户

不修改usb设备固件，仅在主机侧、硬件层之上虚拟成能被不同应用同时识别调用的设备，类似你举例的USB over Ethernet 类软件。

---

### 🤖 DeepSeek

针对你的明确需求：**不修改固件，仅在主机侧，将一个物理USB设备虚拟成两个（或多个）逻辑设备，让不同应用能同时独立访问**。

这是完全可行的，并且比“完整拆分成两个系统设备实例”要简单得多。核心原理是：**在驱动层拦截并分发I/O请求，创建多个虚拟设备节点**。

## 核心实现方案

### 方案一：使用现有软件（无需开发）

| 软件名称 | 原理 | 适用场景 |
|---------|------|---------|
| **Virtual Serial Port Emulator (VSPE)** | 创建虚拟COM口，将物理COM口的数据复制分发到多个虚拟口 | USB转串口设备，多应用同时读写 |
| **com0com + hub4com** | com0com创建配对虚拟串口，hub4com将物理串口绑定到虚拟口并进行数据镜像/分发 | 串口类设备 |
| **USB Network Gate + 自定义桥接** | 创建多个“影子”实例连接到同一远程USB设备 | 局域网共享USB设备 |
| **FlexiHub** | 类似USB over Ethernet，支持同一设备多会话访问 | 跨平台、云环境 |
| **HidHide + Interception** | 先隐藏原始HID设备，再创建多个虚拟HID设备（键盘/鼠标/游戏手柄） | HID类设备（键盘、鼠标、触摸屏、手柄） |

**操作示例（VSPE处理串口设备）：**
```
物理USB串口(COM3) 
    ↓ [VSPE Splitter mode]
    ├→ 虚拟COM4 → 应用A
    └→ 虚拟COM5 → 应用B
```
两个应用同时打开各自COM口，VSPE底层处理数据复制和锁竞争。

### 方案二：开发过滤驱动（精确控制）

如果要完全在“硬件层之上、驱动之下”做到**系统层面呈现两个独立设备节点**，需要开发**过滤驱动(Filter Driver) + 虚拟设备创建**。

**技术路径（Windows）：**

```
物理USB设备
    ↓
USB 总线驱动
    ↓
【你的Upper Filter Driver】 ← 拦截IRP，复制/分发请求
    ├→ 原始功能设备 (PDO1) → 原有驱动栈 → 应用A
    └→ 创建的虚拟设备 (PDO2) → 新建驱动栈 → 应用B
```

**实现要点：**
1. **创建虚拟PDO**：使用`IoCreateDevice`创建一个新的设备对象，附加到系统设备树
2. **IRP分发与镜像**：拦截读/写/IOCTL，复制后发送到原始设备，或将响应分发给多个等待者
3. **处理并发访问**：需要自己实现缓冲区管理、状态同步（原始设备只有一个硬件状态）
4. **处理独占访问**：多数物理USB设备不支持真正的并发（如智能卡、存储设备），需要排队和互斥

**代码框架示例（WDK）**：
```c
// 在Filter Driver的DispatchRead中
NTSTATUS DispatchRead(PDEVICE_OBJECT DeviceObject, PIRP Irp) {
    // 复制IRP
    PIRP mirroredIrp = IoAllocateIrp(OriginalDevice->StackSize, FALSE);
    
    // 转发到原始设备
    IoCallDriver(OriginalDevice, mirroredIrp);
    
    // 原始IRP立即返回（或等待完成）——取决于你需要的同步模型
    // 对于写操作：数据需要发送到硬件一次，但完成通知多个应用
}
```

### 方案三：用户态代理（最简单实用）

**原理：** 运行一个后台服务进程，独占访问物理USB设备，通过**命名管道/Named Pipe、共享内存、Socket**等IPC机制向多个应用提供访问。

```python
# Python示例：代理物理串口给多个客户端
import serial
import socket
import threading

ser = serial.Serial('COM3', 9600)  # 独占打开物理设备

def handle_client(conn):
    while True:
        data = conn.recv(1024)
        ser.write(data)  # 转发到物理设备
        # 广播读取到的数据给所有连接的客户端

server = socket.socket()
server.bind(('127.0.0.1', 8888))
server.listen()

# 广播线程：将串口读取的数据发送给所有连接的客户端
def broadcaster():
    while True:
        data = ser.read()
        for conn in clients:
            conn.send(data)
```

**优点**：
- 不需要内核驱动，无需签名、免重启、无蓝屏风险
- 完全控制并发、缓存、数据流复制规则
- 跨平台（Windows/Linux/macOS）

**缺点**：
- 需要每个应用改用你的IPC接口（或通过虚拟设备驱动中转）
- 对要求底层硬件直接访问的应用（如使用WinUSB的私有协议）不适用

## 方案对比与选择建议

| 场景 | 推荐方案 | 理由 |
|------|---------|------|
| USB转串口，多软件同时读写 | **VSPE (Splitter mode)** | 现成稳定，配置简单 |
| HID设备（键盘、鼠标、刷卡器） | **HidHide + Interception** | 专门解决HID拆分，有现成驱动 |
| 自定义开发，精确控制并发 | **用户态代理服务（方案三）** | 开发成本最低，最灵活 |
| 设备协议复杂，要求系统层面出现两个设备节点 | **开发过滤驱动（方案二）** | 唯一能实现“系统设备管理器可见双设备”的方案 |
| 不需要物理拆分，只做数据复制/记录 | **用户态代理 + 日志/转发** | 最简单，也最稳定 |

## 重要限制（必须了解）

1. **真正的并发不可能**：USB设备只有一个硬件状态机。两个应用同时发送命令时会冲突，你的拆分层必须实现**请求排队、互斥锁**或**数据合并**逻辑。

2. **无法拆分存储设备**：U盘、移动硬盘如果被两个应用同时写，文件系统会崩溃。除非你实现分布式锁/文件级代理（变成网络共享逻辑）。

3. **高速设备不适合软件拆分**：USB摄像头、采集卡的高带宽流无法被软件复制分发（性能不足），应考虑硬件HUB+多个独立设备。

4. **独占控制协议**：某些设备（如智能卡读卡器、特定加密狗）驱动层会锁定设备，无法被第二个程序打开。这种情况下必须用过滤驱动强制绕过。

## 最小工作路径（2小时内实现）

假设你的设备是**USB转串口**：
1. 下载 **Eltima Virtual Serial Port Driver** 或 **VSPE**（免费试用）
2. 创建Splitter：选择物理COM3 → 创建虚拟COM4、COM5
3. 应用A连接COM4，应用B连接COM5 → 完成

如果是**HID设备**（键盘模拟器、刷卡器）：
1. 安装 **Interception** 驱动框架
2. 使用其API创建多个虚拟HID设备，在用户态编写转发逻辑

如果是**其他任意设备**（如自定义USB设备）：
- 最佳选择：**方案三（用户态代理）**，通过反向代理方式让多应用通过IPC访问。

你想针对哪一类设备操作？我可以给出更具体的代码或配置步骤。

---

### 🧑 用户

详细介绍USB over Ethernet 类软件的实现

---

### 🤖 DeepSeek

## USB over Ethernet 类软件的实现原理

这类软件的核心目标是：**将本地USB设备通过网络共享给远程主机，让远程主机像连接本地USB设备一样使用它**。实现方案分为两种架构：**软件USB网关** 和 **硬件USB over IP扩展器**。这里重点讲软件实现。

---

## 一、整体架构

```
【USB设备端（服务器）】                    【客户端】
物理USB设备                                 虚拟USB设备
    ↓                                           ↓
USB驱动栈                                  虚拟USB主机控制器驱动
    ↓                                           ↓
USB over IP服务端 ←─── TCP/IP ───→ USB over IP客户端
 (捕获USB请求)      (网络传输)       (模拟USB设备响应)
```

**两个关键组件：**
1. **服务端**：运行在连接物理USB设备的机器上，负责捕获USB通信
2. **客户端**：运行在远程机器上，创建虚拟USB设备，模拟真实USB设备的响应

---

## 二、核心技术点

### 1. USB请求捕获（服务端）

需要在**USB驱动层**拦截所有USB请求块(URB)。

**Windows实现方案：**

```c
// 使用过滤驱动拦截IRP_MJ_INTERNAL_DEVICE_CONTROL
NTSTATUS DispatchInternalDeviceControl(PDEVICE_OBJECT DeviceObject, PIRP Irp) {
    PIO_STACK_LOCATION stack = IoGetCurrentIrpStackLocation(Irp);
    ULONG ioctl = stack->Parameters.DeviceIoControl.IoControlCode;
    
    switch (ioctl) {
        case IOCTL_INTERNAL_USB_SUBMIT_URB:  // 关键：USB请求
            // 1. 获取URB
            PURB urb = (PURB)stack->Parameters.Others.Argument1;
            
            // 2. 复制URB数据
            USB_REQUEST_PACKET packet;
            memcpy(&packet.urb, urb, sizeof(URB));
            
            // 3. 通过TCP发送到客户端
            SendToRemote(&packet);
            
            // 4. 等待远程响应
            WaitForResponse();
            
            // 5. 将响应数据写回URB
            memcpy(urb, &packet.urb, sizeof(URB));
            
            // 6. 完成IRP
            Irp->IoStatus.Status = STATUS_SUCCESS;
            IoCompleteRequest(Irp, IO_NO_INCREMENT);
            return STATUS_SUCCESS;
    }
    return IoCallDriver(NextDevice, Irp);
}
```

**Linux实现方案：**
```c
// 使用USB/IP协议的内核模块
// drivers/usb/usbip/stub_dev.c
static void stub_rx_loop(struct stub_device *sdev) {
    while (1) {
        // 接收客户端发来的命令
        struct usbip_header header;
        rx_bytes(sdev->ud.socket, &header, sizeof(header));
        
        switch (header.base.command) {
            case USBIP_CMD_SUBMIT:  // USB请求提交
                // 转发到真实USB设备
                usb_submit_urb(sdev->urbs[header.base.seqnum]);
                break;
        }
    }
}
```

### 2. 虚拟USB设备创建（客户端）

在远程系统上创建一个**假的USB设备**，让操作系统相信真的有USB设备插入。

**核心技巧：创建一个虚拟USB主机控制器**

```c
// Windows: 实现虚拟PCI设备作为USB主机控制器
// 参考微软Virtual USB Controller示例

// 1. 创建虚拟PCI设备
NTSTATUS VirtualUSBPciDevice::AddDevice(PDEVICE_OBJECT pdo) {
    // 创建一个虚拟的PCI设备对象
    PDEVICE_OBJECT fdo;
    IoCreateDevice(driver, sizeof(device_extension), 
                   L"\\Device\\VirtualUSBPci", ...);
    
    // 注册为USB主机控制器
    // PCI Class Code: 0x0C0320 (USB 2.0 Host Controller)
    
    // 2. 虚拟Root Hub
    CreateVirtualRootHub();
    
    // 3. 当客户端连接时，动态创建虚拟USB设备
    return STATUS_SUCCESS;
}

// 处理URB请求（来自操作系统）
NTSTATUS OnUrbSubmit(PURB urb) {
    // 将URB通过网络发送到服务端
    SendUrbToServer(urb);
    
    // 等待服务端返回数据，模拟USB设备响应
    WaitForDeviceResponse();
    
    // 完成URB
    return STATUS_SUCCESS;
}
```

**更简单的方案：使用现有USB/IP协议栈**

Linux端直接使用内置的USB/IP模块：
```bash
# 服务端（连接物理设备）
sudo modprobe usbip_host
sudo usbip bind -b 1-2  # 绑定物理USB设备

# 客户端（远程访问）
sudo modprobe vhci-hcd   # 虚拟主机控制器
sudo usbip attach -r server_ip -b 1-2
```

Windows客户端可以使用开源的USB/IP实现（如usbip-win）。

### 3. USB协议网络传输

**关键挑战：** USB是**高频率、低延迟、大数据块**的协议，简单的TCP封装不够。

**实际使用的协议格式（USB/IP规范）：**

```
[USB/IP Header (48 bytes)] + [URB Data (variable)]
```

**USB/IP Header结构：**
```c
struct usbip_header_basic {
    uint32_t command;      // USBIP_CMD_SUBMIT 或 USBIP_RET_SUBMIT
    uint32_t seqnum;       // 序列号，用于请求-响应匹配
    uint32_t devid;        // 设备ID (总线号<<16 | 设备号)
    uint32_t direction;    // USBIP_DIR_OUT 或 USBIP_DIR_IN
    uint32_t ep;           // 端点号
};

struct usbip_header_submit {
    uint32_t transfer_flags;
    int32_t  transfer_buffer_length;
    uint32_t start_frame;
    uint32_t number_of_packets;
    int32_t  interval;
    uint32_t setup_packet[8];  // 控制传输的设置包
};
```

**优化技术：**
- **零拷贝**：使用DMA直接传输USB数据到网络缓冲区
- **批量合并**：多个小URB合并成一个网络包发送
- **TCP_NODELAY**：禁用Nagle算法降低延迟
- **UDP+重传**：对等时传输（如USB音频）改用UDP

### 4. 状态同步与错误处理

**必须处理的特殊场景：**

```python
# 伪代码：处理USB设备拔出
class USBOverIPServer:
    def device_removed_callback(self):
        # 1. 通知所有客户端设备已拔出
        for client in self.clients:
            client.send_event(USBIP_EVENT_DEVICE_GONE)
        
        # 2. 等待2秒（让客户端清理）
        time.sleep(2)
        
        # 3. 关闭所有连接
        for client in self.clients:
            client.close()
    
    def client_disconnect_handler(self):
        # 客户端意外断开，正在等待的URB需要超时
        for pending_urb in self.pending_urbs:
            pending_urb.status = STATUS_DEVICE_NOT_CONNECTED
            pending_urb.complete()
```

## 三、完整数据流示例

以USB鼠标为例，展示一次数据传输：

```
【客户端】（远程电脑）                【服务端】（连接鼠标的电脑）
      |                                        |
  用户移动鼠标                                 |
      |                                        |
OS发送URB查询鼠标状态                         |
      |                                        |
虚拟主机控制器接收URB                         |
      |                                        |
客户端打包URB:                                 |
  - USBIP_HEADER                               |
  - URB数据（查询HID报告）                    |
      |                                        |
     TCP发送 --------------------------------->|
      |                                        |
                                          服务端接收
                                              |
                                          解包URB
                                              |
                                          转发给真实USB鼠标
                                              |
                                          鼠标返回数据（移动偏移量）
                                              |
                                          服务端打包响应
                                              |
     <------------------------------------ TCP返回
      |                                        |
客户端解包响应                                 |
      |                                        |
虚拟主机控制器完成URB                         |
      |                                        |
操作系统收到鼠标数据                           |
      |                                        |
更新光标位置                                   |
```

## 四、高级特性实现

### 1. 多个客户端同时访问一个USB设备

需要实现**虚拟Hub模式**：

```
物理设备
    ↓
服务端（数据复制广播）
    ↓        ↓        ↓
客户端A   客户端B   客户端C
```

**挑战：** USB状态机会冲突。解决方案：
- **只读设备**（如USB摄像头）：直接广播
- **双向设备**（如USB打印机）：使用主从模式，只有一个客户端可以发送命令，其他只接收

### 2. USB设备热插拔

**实现机制（以USB/IP为例）：**

```c
// 服务端：监控USB设备插入/拔出
void usb_monitor_thread() {
    // 监听USB设备事件（通过udev/netlink）
    while (1) {
        if (new_device_detected()) {
            // 广播设备信息
            broadcast_event(USBIP_EVENT_NEW_DEVICE, 
                           bus_id, device_id, descriptors);
            
            // 等待客户端主动attach
        }
        
        if (device_removed()) {
            // 通知所有已attach的设备拔出
            broadcast_event(USBIP_EVENT_REMOVE_DEVICE, bus_id);
        }
    }
}
```

### 3. 性能优化

**零拷贝传输：**
```c
// 使用splice系统调用（Linux）
int pipe_fd[2];
pipe(pipe_fd);

// 从USB设备直接到管道
splice(usb_fd, NULL, pipe_fd[1], NULL, size, SPLICE_F_MOVE);
// 从管道直接到socket
splice(pipe_fd[0], NULL, sock_fd, NULL, size, SPLICE_F_MOVE);
// 全程无CPU拷贝
```

## 五、开源项目参考

如果你想研究完整实现，可以看这些项目：

1. **USB/IP**（Linux内核内置）
   - 代码位置：`drivers/usb/usbip/`
   - 协议规范：`Documentation/usb/usbip_protocol.txt`

2. **usbip-win**（Windows客户端）
   - GitHub: cezanne/usbip-win
   - 实现Windows侧虚拟USB主机控制器

3. **VirtualHere**（商业软件，但有免费版）
   - 支持USB3.0 over IP
   - 实现更高效的压缩和缓冲

4. **usb-redir**（QEMU的USB重定向）
   - 代码位置：`qemu/usb-redir.c`
   - 使用libusb捕获USB请求

## 六、实现难度评估

| 层级 | 所需工作量 | 关键技术 |
|------|-----------|---------|
| **用户态代理**（libusb + TCP） | 1-2周 | libusb捕获，简单协议，只适用于非独占设备 |
| **内核驱动 + TCP** | 2-3月 | WDK/Linux内核编程，URB捕获，虚拟设备创建 |
| **完整USB/IP实现** | 6-12月 | 上述所有 + 热插拔、性能优化、设备类型适配 |
| **商用级产品** | 1年+ | 加上USB3.0支持、加密、压缩、集群、容错 |

**最小可行实现路径：**
1. 使用**libusb**在用户态捕获USB请求（性能差，但最简单）
2. 简单的TCP转发
3. 客户端用**libusb的开发者模式**模拟设备（或者直接写一个上层应用接收数据）

这样你可以在**一周内**得到一个能工作的原型，但仅适用于特定类型的设备（如HID、串口转换器）。

想要我详细展开某个具体部分吗？比如完整的用户态代理实现代码？

---

