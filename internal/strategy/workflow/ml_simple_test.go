package workflow

import (
	"context"
	"testing"
)

func TestMLPipelineBasic(t *testing.T) {
	// 创建ML管道
	mlPipeline := NewMLPipeline("test_strategy")

	// 测试数据收集
	dataRequirements := &DataRequirements{
		HistoryDays:     30,
		MinSamples:      50,
		FeatureTypes:    []string{"price", "volume", "technical"},
		LabelType:       "return",
		ValidationSplit: 0.2,
	}

	ctx := context.Background()
	trainingDataset, err := mlPipeline.CollectTrainingData(ctx, dataRequirements)
	if err != nil {
		t.Fatalf("收集训练数据失败: %v", err)
	}

	if len(trainingDataset.Samples) != dataRequirements.MinSamples {
		t.Errorf("期望样本数 %d，实际 %d", dataRequirements.MinSamples, len(trainingDataset.Samples))
	}

	if len(trainingDataset.FeatureNames) == 0 {
		t.Error("特征名称为空")
	}

	t.Logf("数据收集测试通过: %d 样本, %d 特征", len(trainingDataset.Samples), len(trainingDataset.FeatureNames))

	// 测试模型训练
	modelConfig := &ModelConfig{
		AlgorithmType:              "linear_regression",
		ValidationMethod:           "cross_validation",
		CrossValidationFolds:       3,
		HyperparameterOptimization: false,
	}

	trainedModel, err := mlPipeline.TrainModel(ctx, trainingDataset, modelConfig)
	if err != nil {
		t.Fatalf("模型训练失败: %v", err)
	}

	if trainedModel.Accuracy <= 0 || trainedModel.Accuracy > 1 {
		t.Errorf("模型准确率异常: %f", trainedModel.Accuracy)
	}

	if len(trainedModel.FeatureImportance) == 0 {
		t.Error("特征重要性为空")
	}

	t.Logf("模型训练测试通过: 准确率 %.4f, 特征重要性数量 %d", trainedModel.Accuracy, len(trainedModel.FeatureImportance))

	// 测试模型评估
	modelMetrics, err := mlPipeline.EvaluateModelPerformance(ctx, trainedModel)
	if err != nil {
		t.Fatalf("模型评估失败: %v", err)
	}

	if len(modelMetrics.CrossValidationScores) == 0 {
		t.Error("交叉验证分数为空")
	}

	if len(modelMetrics.LearnedPatterns) == 0 {
		t.Error("学习到的模式为空")
	}

	t.Logf("模型评估测试通过: 验证分数 %.4f, 模式数量 %d", modelMetrics.ValidationScore, len(modelMetrics.LearnedPatterns))

	// 测试参数更新
	err = mlPipeline.UpdateStrategyParameters(ctx, trainedModel)
	if err != nil {
		t.Fatalf("参数更新失败: %v", err)
	}

	t.Log("参数更新测试通过")
}

func TestDifferentModelTypes(t *testing.T) {
	mlPipeline := NewMLPipeline("test_strategy")
	ctx := context.Background()

	// 准备测试数据
	dataRequirements := &DataRequirements{
		HistoryDays:     10,
		MinSamples:      30,
		FeatureTypes:    []string{"price", "volume"},
		LabelType:       "return",
		ValidationSplit: 0.2,
	}

	trainingDataset, err := mlPipeline.CollectTrainingData(ctx, dataRequirements)
	if err != nil {
		t.Fatalf("收集训练数据失败: %v", err)
	}

	// 测试不同的模型类型
	modelTypes := []string{"linear_regression", "random_forest", "ensemble"}

	for _, modelType := range modelTypes {
		t.Run(modelType, func(t *testing.T) {
			modelConfig := &ModelConfig{
				AlgorithmType:              modelType,
				ValidationMethod:           "cross_validation",
				CrossValidationFolds:       3,
				HyperparameterOptimization: false,
			}

			trainedModel, err := mlPipeline.TrainModel(ctx, trainingDataset, modelConfig)
			if err != nil {
				t.Fatalf("模型训练失败 (%s): %v", modelType, err)
			}

			if trainedModel.ModelType != modelType {
				t.Errorf("期望模型类型 %s，实际 %s", modelType, trainedModel.ModelType)
			}

			if trainedModel.Accuracy <= 0 || trainedModel.Accuracy > 1 {
				t.Errorf("模型准确率异常 (%s): %f", modelType, trainedModel.Accuracy)
			}

			t.Logf("模型 %s 训练成功: 准确率 %.4f", modelType, trainedModel.Accuracy)
		})
	}
}