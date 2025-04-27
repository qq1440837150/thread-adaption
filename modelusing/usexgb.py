import sys
import joblib
import xgboost as xgb
import numpy as np

if __name__ == '__main__':
    if len(sys.argv) >= 2:
        # 加载已有的模型
        predictor = joblib.load("D:\\python\\myproject\\reinforcement\\processdata\\maxthreads-model.pkl")

        # 准备输入数据
        X_new = np.array([[int(sys.argv[1])]])

        # 进行预测
        y_pred = predictor.predict(X_new)
        print(y_pred[0])
