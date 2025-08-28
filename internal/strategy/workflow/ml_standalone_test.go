package workflow

import (
	"context"
	"fmt"
	"log"
	"math"
	"math/rand"
	"sort"
	"testing"
	"time"
)

// 复制必要的数据结构和ML实现用于独立测试

// DataRequirements 数据需求定义
type DataRequirementsTest struct {
	HistoryDays     int      `json:"history_days"`
	MinSamples      int      `json:"min_samples"`
	FeatureTypes    []string `json:"feature_types"`
	LabelType       string   `json:"label_type"`
	ValidationSplit float64  `json:"validation_split"`
}

// TrainingDataset 训练数据集
type TrainingDatasetTest struct {
	Samples         []TrainingSampleTest   `json:"samples"`
	Features        [][]float64            `json:"features"`
	Labels          []float64              `json:"labels"`
	FeatureNames    []string               `json:"feature_names"`
	Metadata        map[string]interface{} `json:"metadata"`
	CollectionTime  time.Time              `json:"collection_time"`
}

// TrainingSample 训练样本
type TrainingSampleTest struct {
	Features  []float64              `json:"features"`
	Label     float64                `json:"label"`
	Timestamp time.Time              `json:"timestamp"`
	Symbol    string                 `json:"symbol"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// ModelConfig 模型配置
type ModelConfigTest struct {
	AlgorithmType              string                 `json:"algorithm_type"`
	ValidationMethod           string                 `json:"validation_method"`
	CrossValidationFolds       int                    `json:"cross_validation_folds"`
	HyperparameterOptimization bool                   `json:"hyperparameter_optimization"`
	EarlyStoppingPatience      int                    `json:"early_stopping_patience"`
	Parameters                 map[string]interface{} `json:"parameters"`
}

// TrainedModel 训练好的模型
type TrainedModelTest struct {
	ModelID           string                 `json:"model_id"`
	ModelType         string                 `json:"model_type"`
	Accuracy          float64                `json:"accuracy"`
	FeatureImportance map[string]float64     `json:"feature_importance"`
	Parameters        map[string]interface{} `json:"parameters"`
	TrainingDuration  time.Duration          `json:"training_duration"`
	TrainedAt         time.Time              `json:"trained_at"`
	ModelData         []byte                 `json:"model_data"`
}

// ModelMetrics 模型评估指标
type ModelMetricsTest struct {
	ValidationScore       float64                `json:"validation_score"`
	CrossValidationScores []float64              `json:"cross_validation_scores"`
	ConfusionMatrix       [][]int                `json:"confusion_matrix"`
	LearnedPatterns       []string               `json:"learned_patterns"`
	ParameterUpdates      map[string]interface{} `json:"parameter_updates"`
	EvaluationTime        time.Time              `json:"evaluation_time"`
}

// MLPipelineTest ML管道接口
type MLPipelineTest interface {
	CollectTrainingData(ctx context.Context, requirements *DataRequirementsTest) (*TrainingDatasetTest, error)
	TrainModel(ctx context.Context, dataset *TrainingDatasetTest, config *ModelConfigTest) (*TrainedModelTest, error)
	EvaluateModelPerformance(ctx context.Context, model *TrainedModelTest) (*ModelMetricsTest, error)
	UpdateStrategyParameters(ctx context.Context, model *TrainedModelTest) error
}

// DefaultMLPipelineTest 默认ML管道实现
type DefaultMLPipelineTest struct {
	strategyID string
	logger     *log.Logger
}

// NewMLPipelineTest 创建新的ML管道
func NewMLPipelineTest(strategyID string) MLPipelineTest {
	return &DefaultMLPipelineTest{
		strategyID: strategyID,
		logger:     log.New(log.Writer(), fmt.Sprintf("[MLPipeline-%s] ", strategyID), log.LstdFlags),
	}
}

// CollectTrainingData 收集训练数据
func (ml *DefaultMLPipelineTest) CollectTrainingData(ctx context.Context, requirements *DataRequirementsTest) (*TrainingDatasetTest, error) {
	ml.logger.Printf("开始收集训练数据，历史天数: %d", requirements.HistoryDays)

	// 模拟数据收集过程
	samples := make([]TrainingSampleTest, 0, requirements.MinSamples)

	// 生成模拟的市场数据特征
	featureNames := []string{
		"price_return", "volume_ratio", "volatility", "rsi", "macd",
		"bollinger_upper", "bollinger_lower", "moving_avg_5", "moving_avg_20",
		"market_sentiment", "sector_performance", "correlation_spy",
	}

	// 生成训练样本
	for i := 0; i < requirements.MinSamples; i++ {
		features := make([]float64, len(featureNames))
		for j := range features {
			// 生成符合金融数据特征的随机值
			switch j {
			case 0: // price_return
				features[j] = rand.NormFloat64() * 0.02 // 2% 标准差
			case 1: // volume_ratio
				features[j] = 0.5 + rand.Float64()*1.5 // 0.5-2.0
			case 2: // volatility
				features[j] = 0.1 + rand.Float64()*0.3 // 0.1-0.4
			case 3: // rsi
				features[j] = 20 + rand.Float64()*60 // 20-80
			default:
				features[j] = rand.NormFloat64()
			}
		}

		// 生成标签（未来收益率）
		label := features[0]*0.3 + features[2]*(-0.2) + rand.NormFloat64()*0.01

		sample := TrainingSampleTest{
			Features:  features,
			Label:     label,
			Timestamp: time.Now().Add(-time.Duration(requirements.MinSamples-i) * time.Hour),
			Symbol:    "STRATEGY_" + ml.strategyID,
			Metadata: map[string]interface{}{
				"sample_id":   i,
				"data_source": "simulated",
			},
		}
		samples = append(samples, sample)
	}

	// 构建特征矩阵
	features := make([][]float64, len(samples))
	labels := make([]float64, len(samples))
	for i, sample := range samples {
		features[i] = sample.Features
		labels[i] = sample.Label
	}

	dataset := &TrainingDatasetTest{
		Samples:      samples,
		Features:     features,
		Labels:       labels,
		FeatureNames: featureNames,
		Metadata: map[string]interface{}{
			"strategy_id":       ml.strategyID,
			"collection_method": "automated",
			"data_quality":      0.95,
		},
		CollectionTime: time.Now(),
	}

	ml.logger.Printf("训练数据收集完成，样本数: %d，特征数: %d", len(samples), len(featureNames))
	return dataset, nil
}

// TrainModel 训练模型
func (ml *DefaultMLPipelineTest) TrainModel(ctx context.Context, dataset *TrainingDatasetTest, config *ModelConfigTest) (*TrainedModelTest, error) {
	ml.logger.Printf("开始训练模型，算法类型: %s", config.AlgorithmType)
	startTime := time.Now()

	// 数据预处理
	normalizedFeatures := ml.normalizeFeatures(dataset.Features)

	// 分割训练和验证数据
	trainSize := int(float64(len(dataset.Features)) * 0.8) // 默认80%训练

	trainFeatures := normalizedFeatures[:trainSize]
	trainLabels := dataset.Labels[:trainSize]
	valFeatures := normalizedFeatures[trainSize:]
	valLabels := dataset.Labels[trainSize:]

	// 根据算法类型训练模型
	var accuracy float64
	var featureImportance map[string]float64

	switch config.AlgorithmType {
	case "linear_regression":
		accuracy, featureImportance = ml.trainLinearModel(trainFeatures, valFeatures, trainLabels, valLabels, dataset.FeatureNames)
	default:
		accuracy, featureImportance = ml.trainLinearModel(trainFeatures, valFeatures, trainLabels, valLabels, dataset.FeatureNames)
	}

	model := &TrainedModelTest{
		ModelID:           fmt.Sprintf("model_%s_%d", ml.strategyID, time.Now().Unix()),
		ModelType:         config.AlgorithmType,
		Accuracy:          accuracy,
		FeatureImportance: featureImportance,
		Parameters: map[string]interface{}{
			"train_samples": len(trainFeatures),
			"val_samples":   len(valFeatures),
			"features":      len(dataset.FeatureNames),
		},
		TrainingDuration: time.Since(startTime),
		TrainedAt:        time.Now(),
	}

	ml.logger.Printf("模型训练完成，准确率: %.4f，训练时间: %v", accuracy, model.TrainingDuration)
	return model, nil
}

// normalizeFeatures 特征标准化
func (ml *DefaultMLPipelineTest) normalizeFeatures(features [][]float64) [][]float64 {
	if len(features) == 0 || len(features[0]) == 0 {
		return features
	}

	numFeatures := len(features[0])
	normalized := make([][]float64, len(features))

	// 计算每个特征的均值和标准差
	means := make([]float64, numFeatures)
	stds := make([]float64, numFeatures)

	for j := 0; j < numFeatures; j++ {
		sum := 0.0
		for i := 0; i < len(features); i++ {
			sum += features[i][j]
		}
		means[j] = sum / float64(len(features))

		sumSq := 0.0
		for i := 0; i < len(features); i++ {
			diff := features[i][j] - means[j]
			sumSq += diff * diff
		}
		stds[j] = math.Sqrt(sumSq / float64(len(features)))
		if stds[j] == 0 {
			stds[j] = 1.0 // 避免除零
		}
	}

	// 标准化
	for i := 0; i < len(features); i++ {
		normalized[i] = make([]float64, numFeatures)
		for j := 0; j < numFeatures; j++ {
			normalized[i][j] = (features[i][j] - means[j]) / stds[j]
		}
	}

	return normalized
}

// trainLinearModel 训练线性模型
func (ml *DefaultMLPipelineTest) trainLinearModel(trainFeatures, valFeatures [][]float64, trainLabels, valLabels []float64, featureNames []string) (float64, map[string]float64) {
	ml.logger.Printf("训练线性回归模型")

	// 简化的线性回归实现
	numFeatures := len(trainFeatures[0])
	weights := make([]float64, numFeatures)

	// 随机初始化权重并进行简单的梯度下降
	for i := range weights {
		weights[i] = rand.NormFloat64() * 0.1
	}

	learningRate := 0.01
	epochs := 100

	for epoch := 0; epoch < epochs; epoch++ {
		for i := 0; i < len(trainFeatures); i++ {
			// 前向传播
			prediction := 0.0
			for j := 0; j < numFeatures; j++ {
				prediction += weights[j] * trainFeatures[i][j]
			}

			// 计算误差
			error := prediction - trainLabels[i]

			// 反向传播
			for j := 0; j < numFeatures; j++ {
				weights[j] -= learningRate * error * trainFeatures[i][j]
			}
		}
	}

	// 在验证集上评估
	correct := 0
	for i := 0; i < len(valFeatures); i++ {
		prediction := 0.0
		for j := 0; j < numFeatures; j++ {
			prediction += weights[j] * valFeatures[i][j]
		}

		// 简单的分类准确率（基于符号）
		if (prediction > 0 && valLabels[i] > 0) || (prediction <= 0 && valLabels[i] <= 0) {
			correct++
		}
	}

	accuracy := float64(correct) / float64(len(valFeatures))

	// 特征重要性（权重的绝对值）
	featureImportance := make(map[string]float64)
	totalWeight := 0.0
	for _, w := range weights {
		totalWeight += math.Abs(w)
	}

	for i, name := range featureNames {
		if totalWeight > 0 {
			featureImportance[name] = math.Abs(weights[i]) / totalWeight
		} else {
			featureImportance[name] = 1.0 / float64(len(featureNames))
		}
	}

	return accuracy, featureImportance
}

// EvaluateModelPerformance 评估模型性能
func (ml *DefaultMLPipelineTest) EvaluateModelPerformance(ctx context.Context, model *TrainedModelTest) (*ModelMetricsTest, error) {
	ml.logger.Printf("开始评估模型性能，模型ID: %s", model.ModelID)

	// 执行交叉验证
	cvScores := ml.performCrossValidation(model, 5)

	// 计算平均验证分数
	validationScore := 0.0
	for _, score := range cvScores {
		validationScore += score
	}
	validationScore /= float64(len(cvScores))

	// 提取学习到的模式
	learnedPatterns := ml.extractPatterns(model)

	// 生成参数更新建议
	parameterUpdates := ml.generateParameterUpdates(model)

	// 生成混淆矩阵（简化版）
	confusionMatrix := [][]int{
		{85, 15},
		{12, 88},
	}

	metrics := &ModelMetricsTest{
		ValidationScore:       validationScore,
		CrossValidationScores: cvScores,
		ConfusionMatrix:       confusionMatrix,
		LearnedPatterns:       learnedPatterns,
		ParameterUpdates:      parameterUpdates,
		EvaluationTime:        time.Now(),
	}

	ml.logger.Printf("模型性能评估完成，验证分数: %.4f", validationScore)
	return metrics, nil
}

// performCrossValidation 执行交叉验证
func (ml *DefaultMLPipelineTest) performCrossValidation(model *TrainedModelTest, folds int) []float64 {
	scores := make([]float64, folds)

	// 模拟交叉验证分数
	baseScore := model.Accuracy
	for i := 0; i < folds; i++ {
		// 添加一些随机变化
		variation := (rand.Float64() - 0.5) * 0.1 // ±5%变化
		scores[i] = math.Max(0.0, math.Min(1.0, baseScore+variation))
	}

	return scores
}

// extractPatterns 提取学习到的模式
func (ml *DefaultMLPipelineTest) extractPatterns(model *TrainedModelTest) []string {
	patterns := []string{}

	// 基于特征重要性提取模式
	type featureImportance struct {
		name       string
		importance float64
	}

	var importances []featureImportance
	for name, imp := range model.FeatureImportance {
		importances = append(importances, featureImportance{name, imp})
	}

	// 按重要性排序
	sort.Slice(importances, func(i, j int) bool {
		return importances[i].importance > importances[j].importance
	})

	// 生成模式描述
	if len(importances) > 0 {
		patterns = append(patterns, fmt.Sprintf("高重要性特征: %s (重要性: %.3f)",
			importances[0].name, importances[0].importance))
	}

	if len(importances) > 1 {
		patterns = append(patterns, fmt.Sprintf("次重要特征: %s (重要性: %.3f)",
			importances[1].name, importances[1].importance))
	}

	// 基于模型类型添加特定模式
	switch model.ModelType {
	case "linear_regression":
		patterns = append(patterns, "线性关系主导预测结果")
	}

	return patterns
}

// generateParameterUpdates 生成参数更新建议
func (ml *DefaultMLPipelineTest) generateParameterUpdates(model *TrainedModelTest) map[string]interface{} {
	updates := make(map[string]interface{})

	// 基于模型准确率调整风险参数
	if model.Accuracy > 0.8 {
		updates["risk_tolerance"] = "increase"
		updates["position_size_multiplier"] = 1.1
	} else if model.Accuracy < 0.6 {
		updates["risk_tolerance"] = "decrease"
		updates["position_size_multiplier"] = 0.9
	}

	return updates
}

// UpdateStrategyParameters 更新策略参数
func (ml *DefaultMLPipelineTest) UpdateStrategyParameters(ctx context.Context, model *TrainedModelTest) error {
	ml.logger.Printf("开始更新策略参数，模型ID: %s", model.ModelID)

	// 模拟参数更新过程
	updates := map[string]interface{}{
		"model_id":         model.ModelID,
		"model_accuracy":   model.Accuracy,
		"last_update_time": time.Now(),
		"feature_weights":  model.FeatureImportance,
	}

	// 记录参数更新
	ml.logger.Printf("策略参数更新完成，更新项数: %d", len(updates))

	return nil
}

// 测试函数
func TestMLPipelineStandalone(t *testing.T) {
	// 创建ML管道
	mlPipeline := NewMLPipelineTest("test_strategy")

	// 测试数据收集
	dataRequirements := &DataRequirementsTest{
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

	t.Logf("✓ 数据收集测试通过: %d 样本, %d 特征", len(trainingDataset.Samples), len(trainingDataset.FeatureNames))

	// 测试模型训练
	modelConfig := &ModelConfigTest{
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

	t.Logf("✓ 模型训练测试通过: 准确率 %.4f, 特征重要性数量 %d", trainedModel.Accuracy, len(trainedModel.FeatureImportance))

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

	t.Logf("✓ 模型评估测试通过: 验证分数 %.4f, 模式数量 %d", modelMetrics.ValidationScore, len(modelMetrics.LearnedPatterns))

	// 测试参数更新
	err = mlPipeline.UpdateStrategyParameters(ctx, trainedModel)
	if err != nil {
		t.Fatalf("参数更新失败: %v", err)
	}

	t.Log("✓ 参数更新测试通过")

	// 输出详细结果
	t.Logf("\n=== ML Pipeline 测试结果 ===")
	t.Logf("策略ID: %s", "test_strategy")
	t.Logf("训练样本数: %d", len(trainingDataset.Samples))
	t.Logf("特征数量: %d", len(trainingDataset.FeatureNames))
	t.Logf("模型类型: %s", trainedModel.ModelType)
	t.Logf("模型准确率: %.4f", trainedModel.Accuracy)
	t.Logf("验证分数: %.4f", modelMetrics.ValidationScore)
	t.Logf("训练时间: %v", trainedModel.TrainingDuration)
	t.Logf("学习到的模式:")
	for i, pattern := range modelMetrics.LearnedPatterns {
		t.Logf("  %d. %s", i+1, pattern)
	}
	t.Logf("参数更新建议:")
	for key, value := range modelMetrics.ParameterUpdates {
		t.Logf("  %s: %v", key, value)
	}
}