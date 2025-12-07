# 馃殤 KubeHealer 璇婃柇鎶ュ憡: oom-pod

> 鐢熸垚鏃堕棿: 2025-12-07 14:58:59

## 1. 鍩虹淇℃伅

| 鎸囨爣 | 鍊?|
| :--- | :--- |
| **Pod 鍚嶇О** | `oom-pod` |
| **鍛藉悕绌洪棿** | `default` |
| **鎵€鍦ㄨ妭鐐?* | `minikube` |
| **褰撳墠鐘舵€?* | **Running** |
| **閲嶅惎娆℃暟** | 83 |

## 2. 瀹瑰櫒娣卞害鍒嗘瀽

### 馃洃 瀹瑰櫒: memory-eater

- **鐘舵€?*: Waiting
- **璧勬簮閰嶇疆**: `CPU(Req=鏈缃?Lim=鏈缃? | Mem(Req=50Mi/Lim=100Mi)`
- **鍘熷洜**: CrashLoopBackOff
- **璇︾粏淇℃伅**: back-off 5m0s restarting failed container=memory-eater pod=oom-pod_default(97f12ee0-0d6b-47ca-820e-b85f612108a4)

**馃攳 璇婃柇鍙戠幇:**

> 馃洃 **鍐呭瓨婧㈠嚭 (OOMKilled)**
> *鍘熷鎶ラ敊: Exit Code: 1 (General Error (搴旂敤鍐呴儴閿欒))*
> **馃挕 淇寤鸿**: 妫€娴嬪埌鍐呭瓨闄愬埗 Limit=100Mi锛屽缓璁€傚綋璋冨ぇ
>

---

## 3. 鏈€杩戜簨浠?(Events)

- 馃敼 [5鍒嗛挓鍓峕 Pulling: Pulling image "polinux/stress"
- 馃敻 [3鍒嗛挓鍓峕 BackOff: Back-off restarting failed container memory-eater in pod oom-pod_default(97f12ee0-0d6b-47ca-820e-b85f612108a4)


馃弫 [PID: 14452] 璇婃柇缁撴潫锛岀▼搴忓嵆灏嗛€€鍑恒€?
