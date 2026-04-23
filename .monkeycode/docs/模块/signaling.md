# 信令服务模块

## 概述

信令服务模块 (`internal/signaling/`) 处理 WebRTC 通话的信令转发，是建立 P2P 连接的关键组件。

## 功能

1. **房间管理**
   - 创建房间
   - 删除房间
   - 查询房间

2. **用户管理**
   - 用户加入房间
   - 用户离开房间
   - 成员列表

3. **信令转发**
   - Offer/Answer 转发
   - ICE Candidate 转发

4. **清理机制**
   - 定时清理空闲房间
   - 断开连接标记

## Server 结构

```go
type Server struct {
    cfg    Config
    logger Logger
    rooms  map[string]*Room
    mu     sync.RWMutex
}

type Room struct {
    ID        string
    Name      string
    OwnerID   string
    Members   map[string]*Peer
    CreatedAt time.Time
    mu        sync.RWMutex
}

type Peer struct {
    ID        string
    UserID    string
    RoomID    string
    Connected bool
    JoinedAt  time.Time
}
```

## 核心方法

### 房间管理

```go
// 创建房间
func (s *Server) CreateRoom(roomID, name, ownerID string) *Room

// 获取房间
func (s *Server) GetRoom(roomID string) (*Room, bool)

// 删除房间
func (s *Server) DeleteRoom(roomID string)
```

### 成员管理

```go
// 加入房间
func (s *Server) JoinRoom(roomID, peerID, userID string) (*Peer, bool)

// 离开房间
func (s *Server) LeaveRoom(roomID, peerID string)

// 获取成员
func (s *Server) GetPeers(roomID string) []*Peer

// 成员数量
func (s *Server) RoomMemberCount(roomID string) int
```

## 清理机制

信令服务器每 30 秒检查一次房间状态：

1. **空闲清理**: 如果房间内无已连接成员，删除房间
2. **超时清理**: 超过 24 小时且无成员的房间被删除

```go
func (s *Server) cleanupRooms() {
    // 检查每个房间
    // 断开所有连接的 peer
    // 删除无成员的空闲房间
    // 删除超时房间
}
```

## 与 WebSocket 集成

信令服务器与 WebSocket 服务器配合工作：

```
Client A                   WebSocket Server              Signaling Server
   |                            |                              |
   |--- offer --->             |                              |
   |                            |--- forward offer --->       |
   |                            |                              |
   |                            |<-- forward offer ----        |
   |<-- answer ---             |                              |
   |                            |                              |
   |<-- ice_candidate -------- |<-- ice_candidate -----------  |
   |--- ice_candidate ------> |                              |
```

## 消息流

1. 用户加入房间时，WebSocket 通知信令服务器
2. 信令服务器更新房间成员状态
3. 用户离开时，清理成员信息
4. 定时任务清理无效房间

## 配置

```go
type Config struct {
    Port int  // 信令服务器端口
}
```

## 注意事项

1. **线程安全**: 使用 RWMutex 保护共享数据
2. **状态同步**: 信令状态需与房间服务状态同步
3. **超时处理**: 合理设置清理间隔
4. **日志记录**: 记录关键操作便于调试
