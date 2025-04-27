import numpy as np
import pandas as pd
import xgboost as xgb
import joblib


def printMaxThreads(df_users):
    # 找到数据集中的最低和最高工作负载
    min_workload = 10
    max_workload = 200
    # 初始化变量
    w0 = min_workload
    crossover_point = None
    confidence_lower_bound = 0.96
    # 迭代增加工作负载
    while w0 <= max_workload:
        # 计算当前工作负载及以下的所有SLO满足度值
        current_data = df_users[(df_users["threads"] > w0-1) & (df_users["threads"] <= w0)].iloc[:, 2] / df_users[(df_users["threads"] >w0-1) &(df_users["threads"] <= w0)].iloc[:, 3]
        # 计算95%置信区间
        mean_sat = np.mean(current_data)
        std_dev = np.std(current_data, ddof=1)
        n = len(current_data)
        confidence_interval = (mean_sat - 1.96 * (std_dev / np.sqrt(n)), mean_sat + 1.96 * (std_dev / np.sqrt(n)))

        # 检查置信区间的下限是否低于90%
        if confidence_interval[0] < confidence_lower_bound:
            crossover_point = w0
            break

        # 逐步增加工作负载
        w0 += 1  # 这里需要根据实际情况调整步长
    return crossover_point


df = pd.read_csv("output2-53120.csv")

df_gp = df.groupby(['users', 'qos', 'threads'])['gp'].mean().reset_index()
df_all = df.groupby(['users', 'qos', 'threads'])['all'].mean().reset_index()
X = []
y = []
for i in range(1501, 5002, 100):
    dfgp_1500 = df_gp[df_gp["users"] == i]
    dfall_1500 = df_all[df_all["users"] == i]
    df_users = df[(df["users"] == i) & (df["gp"] > 10)]
    print(i)
    maxThreads = printMaxThreads(df_users)
    print(maxThreads)
    X.append(i)
    y.append(maxThreads)
X = np.array(X).reshape(-1, 1)
y = np.array(y).reshape(-1, 1)
# 构建XGBoost分类器
clf = xgb.XGBRegressor()
# 训练模型
clf.fit(X, y)
# 进行预测
y_pred = clf.predict([3602])
print(y_pred)
joblib.dump(clf, "maxthreads-model.pkl")
