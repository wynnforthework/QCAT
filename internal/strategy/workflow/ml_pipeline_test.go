package workflow

import (
	"context"
	"testing"
	"time"
)

func TestMLPipelineIntegration(t *testing.T) {
	// 创建ML管道
	mlPipeline := NewMLPipeline("test_strategy")

	// 测试数据收集
	dataRequirements := &DataRequirements{
		HistoryDays:     30,
		MinSamples:      100,
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

	// 测试模型训练
	modelConfig := &ModelConfig{
		AlgorithmType:              "ensemble",
		ValidationMethod:           "cross_validation",
		CrossValidationFolds:       3,
		HyperparameterOptimization: true,
	}

	trainedModel, err := mlPipeline.TrainModel(ctx, trainingDataset, modelConfig)
	if err != nil {
		t.Fatalf("模型训练失败: %v", err)
	}

	if trainedModel.Accuracy <= 0 || trainedModel.Accuracy > 1 {
		t.Errorf("模型准确率异常: %f", trainedModel.Accuracy)
	}

	// 测试模型评估
	modelMetrics, err := mlPipeline.EvaluateModelPerformance(ctx, trainedModel)
	if err != nil {
		t.Fatalf("模型评估失败: %v", err)
	}

	if len(modelMetrics.CrossValidationScores) == 0 {
		t.Error("交叉验证分数为空")
	}

	// 测试参数更新
	err = mlPipeline.UpdateStrategyParameters(ctx, trainedModel)
	if err != nil {
		t.Fatalf("参数更新失败: %v", err)
	}

	t.Logf("ML管道测试完成:")
	t.Logf("- 训练样本数: %d", len(trainingDataset.Samples))
	t.Logf("- 模型类型: %s", trainedModel.ModelType)
	t.Logf("- 模型准确率: %.4f", trainedModel.Accuracy)
	t.Logf("- 验证分数: %.4f", modelMetrics.ValidationScore)
	t.Logf("- 学习到的模式数: %d", len(modelMetrics.LearnedPatterns))
}

func TestStrategyWorkflowEngineWithRealML(t *testing.T) {
	// 创建策略工作流引擎
	config := GetDefaultWorkflowConfig()
	config.MaxConcurrentJobs = 2
	config.LearningTimeout = 30 * time.Second

	engine := NewStrategyWorkflowEngine("test_strategy_ml", "测试ML策略", config)

	// 启动引擎
	err := engine.Start()
	if err != nil {
		t.Fatalf("启动引擎失败: %v", err)
	}
	defer engine.Stop()

	// 执行学习阶段
	err = engine.executeLearningStage()
	if err != nil {
		t.Fatalf("执行学习阶段失败: %v", err)
	}

	// 检查任务历史
	history := engine.GetJobHistory()
	if len(history) == 0 {
		t.Error("没有找到学习任务历史")
	}

	learningJob := history[len(history)-1]
	if learningJob.Type != JobLearning {
		t.Errorf("期望任务类型 %v，实际 %v", JobLearning, learningJob.Type)
	}

	if learningJob.Status != JobCompleted {
		t.Errorf("期望任务状态 %v，实际 %v", JobCompleted, learningJob.Status)
	}

	// 检查学习结果
	result, ok := learningJob.Result.(map[string]interface{})
	if !ok {
		t.Fatal("学习结果格式错误")
	}

	if _, exists := result["model_accuracy"]; !exists {
		t.Error("学习结果中缺少模型准确率")
	}

	if _, exists := result["feature_importance"]; !exists {
		t.Error("学习结果中缺少特征重要性")
	}

	t.Logf("策略工作流ML测试完成:")
	t.Logf("- 策略ID: %s", engine.StrategyID)
	t.Logf("- 当前阶段: %s", engine.GetCurrentStage().String())
	t.Logf("- 学习任务耗时: %v", learningJob.EndTime.Sub(learningJob.StartTime))
}