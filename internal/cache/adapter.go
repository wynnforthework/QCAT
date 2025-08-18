package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

// CacheAdapter adapts the CacheManager to implement the Cacher interface
type CacheAdapter struct {
	manager *CacheManager
}

// NewCacheAdapter creates a new cache adapter
func NewCacheAdapter(manager *CacheManager) *CacheAdapter {
	return &CacheAdapter{
		manager: manager,
	}
}

// Get retrieves a value and unmarshals it into dest
func (ca *CacheAdapter) Get(ctx context.Context, key string, dest interface{}) error {
	// Create a temporary interface{} to hold the raw value
	var rawValue interface{}
	err := ca.manager.Get(ctx, key, &rawValue)
	if err != nil {
		return err
	}

	// If dest is a pointer to interface{}, just assign the value
	if destPtr, ok := dest.(*interface{}); ok {
		*destPtr = rawValue
		return nil
	}

	// Try to unmarshal JSON if value is a string
	if valueStr, ok := rawValue.(string); ok {
		return json.Unmarshal([]byte(valueStr), dest)
	}

	// Try to marshal and unmarshal to convert types
	valueBytes, err := json.Marshal(rawValue)
	if err != nil {
		return fmt.Errorf("failed to marshal cached value: %w", err)
	}

	return json.Unmarshal(valueBytes, dest)
}

// Set stores a value with expiration
func (ca *CacheAdapter) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return ca.manager.Set(ctx, key, value, expiration)
}

// Delete removes a key
func (ca *CacheAdapter) Delete(ctx context.Context, key string) error {
	return ca.manager.Delete(ctx, key)
}

// Exists checks if a key exists
func (ca *CacheAdapter) Exists(ctx context.Context, key string) (bool, error) {
	return ca.manager.Exists(ctx, key)
}

// HGet retrieves a hash field
func (ca *CacheAdapter) HGet(ctx context.Context, key, field string, dest interface{}) error {
	hashKey := fmt.Sprintf("%s:%s", key, field)
	return ca.Get(ctx, hashKey, dest)
}

// HSet sets a hash field
func (ca *CacheAdapter) HSet(ctx context.Context, key, field string, value interface{}) error {
	hashKey := fmt.Sprintf("%s:%s", key, field)
	return ca.Set(ctx, hashKey, value, time.Hour) // Default 1 hour expiration
}

// HGetAll retrieves all hash fields
func (ca *CacheAdapter) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	// This is a simplified implementation
	// In a real implementation, you might want to store hash structure differently
	var value interface{}
	err := ca.manager.Get(ctx, key, &value)
	if err != nil {
		return nil, err
	}

	if hashMap, ok := value.(map[string]string); ok {
		return hashMap, nil
	}

	// Try to convert from map[string]interface{}
	if hashMapInterface, ok := value.(map[string]interface{}); ok {
		result := make(map[string]string)
		for k, v := range hashMapInterface {
			result[k] = fmt.Sprintf("%v", v)
		}
		return result, nil
	}

	return nil, fmt.Errorf("value is not a hash map")
}

// HDel deletes hash fields
func (ca *CacheAdapter) HDel(ctx context.Context, key string, fields ...string) error {
	for _, field := range fields {
		hashKey := fmt.Sprintf("%s:%s", key, field)
		if err := ca.Delete(ctx, hashKey); err != nil {
			return err
		}
	}
	return nil
}

// LPush pushes values to the left of a list
func (ca *CacheAdapter) LPush(ctx context.Context, key string, values ...interface{}) error {
	// Simplified implementation - store as JSON array
	existingList, err := ca.getList(ctx, key)
	if err != nil && err.Error() != fmt.Sprintf("cache miss: key %s not found in any cache layer", key) {
		return err
	}

	// Prepend new values
	newList := append(values, existingList...)
	return ca.Set(ctx, key, newList, time.Hour)
}

// RPush pushes values to the right of a list
func (ca *CacheAdapter) RPush(ctx context.Context, key string, values ...interface{}) error {
	existingList, err := ca.getList(ctx, key)
	if err != nil && err.Error() != fmt.Sprintf("cache miss: key %s not found in any cache layer", key) {
		return err
	}

	// Append new values
	newList := append(existingList, values...)
	return ca.Set(ctx, key, newList, time.Hour)
}

// LPop pops a value from the left of a list
func (ca *CacheAdapter) LPop(ctx context.Context, key string, dest interface{}) error {
	list, err := ca.getList(ctx, key)
	if err != nil {
		return err
	}

	if len(list) == 0 {
		return fmt.Errorf("list is empty")
	}

	// Get first element
	value := list[0]
	
	// Update list without first element
	newList := list[1:]
	if err := ca.Set(ctx, key, newList, time.Hour); err != nil {
		return err
	}

	// Convert value to dest
	return ca.convertValue(value, dest)
}

// RPop pops a value from the right of a list
func (ca *CacheAdapter) RPop(ctx context.Context, key string, dest interface{}) error {
	list, err := ca.getList(ctx, key)
	if err != nil {
		return err
	}

	if len(list) == 0 {
		return fmt.Errorf("list is empty")
	}

	// Get last element
	value := list[len(list)-1]
	
	// Update list without last element
	newList := list[:len(list)-1]
	if err := ca.Set(ctx, key, newList, time.Hour); err != nil {
		return err
	}

	// Convert value to dest
	return ca.convertValue(value, dest)
}

// LRange returns a range of list elements
func (ca *CacheAdapter) LRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	list, err := ca.getList(ctx, key)
	if err != nil {
		return nil, err
	}

	// Handle negative indices
	length := int64(len(list))
	if start < 0 {
		start = length + start
	}
	if stop < 0 {
		stop = length + stop
	}

	// Bounds checking
	if start < 0 {
		start = 0
	}
	if stop >= length {
		stop = length - 1
	}
	if start > stop {
		return []string{}, nil
	}

	// Convert to strings
	result := make([]string, 0, stop-start+1)
	for i := start; i <= stop; i++ {
		result = append(result, fmt.Sprintf("%v", list[i]))
	}

	return result, nil
}

// SAdd adds members to a set
func (ca *CacheAdapter) SAdd(ctx context.Context, key string, members ...interface{}) error {
	existingSet, err := ca.getSet(ctx, key)
	if err != nil && err.Error() != fmt.Sprintf("cache miss: key %s not found in any cache layer", key) {
		return err
	}

	// Add new members
	for _, member := range members {
		memberStr := fmt.Sprintf("%v", member)
		existingSet[memberStr] = true
	}

	return ca.Set(ctx, key, existingSet, time.Hour)
}

// SRem removes members from a set
func (ca *CacheAdapter) SRem(ctx context.Context, key string, members ...interface{}) error {
	existingSet, err := ca.getSet(ctx, key)
	if err != nil {
		return err
	}

	// Remove members
	for _, member := range members {
		memberStr := fmt.Sprintf("%v", member)
		delete(existingSet, memberStr)
	}

	return ca.Set(ctx, key, existingSet, time.Hour)
}

// SMembers returns all members of a set
func (ca *CacheAdapter) SMembers(ctx context.Context, key string) ([]string, error) {
	set, err := ca.getSet(ctx, key)
	if err != nil {
		return nil, err
	}

	members := make([]string, 0, len(set))
	for member := range set {
		members = append(members, member)
	}

	return members, nil
}

// SIsMember checks if a member exists in a set
func (ca *CacheAdapter) SIsMember(ctx context.Context, key string, member interface{}) (bool, error) {
	set, err := ca.getSet(ctx, key)
	if err != nil {
		return false, err
	}

	memberStr := fmt.Sprintf("%v", member)
	return set[memberStr], nil
}

// ZAdd adds a member with score to a sorted set
func (ca *CacheAdapter) ZAdd(ctx context.Context, key string, score float64, member interface{}) error {
	// Simplified implementation using a map
	zsetKey := fmt.Sprintf("zset:%s", key)
	memberStr := fmt.Sprintf("%v", member)
	scoreKey := fmt.Sprintf("%s:score:%s", zsetKey, memberStr)
	
	// Store the score
	if err := ca.Set(ctx, scoreKey, score, time.Hour); err != nil {
		return err
	}

	// Add member to the set
	return ca.SAdd(ctx, zsetKey, member)
}

// ZRange returns members in a sorted set by rank
func (ca *CacheAdapter) ZRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	// Simplified implementation - just return set members
	zsetKey := fmt.Sprintf("zset:%s", key)
	return ca.SMembers(ctx, zsetKey)
}

// ZRangeByScore returns members in a sorted set by score range
func (ca *CacheAdapter) ZRangeByScore(ctx context.Context, key string, min, max string) ([]string, error) {
	// Simplified implementation - return all members
	zsetKey := fmt.Sprintf("zset:%s", key)
	return ca.SMembers(ctx, zsetKey)
}

// ZRem removes members from a sorted set
func (ca *CacheAdapter) ZRem(ctx context.Context, key string, members ...interface{}) error {
	zsetKey := fmt.Sprintf("zset:%s", key)
	
	// Remove from set
	if err := ca.SRem(ctx, zsetKey, members...); err != nil {
		return err
	}

	// Remove scores
	for _, member := range members {
		memberStr := fmt.Sprintf("%v", member)
		scoreKey := fmt.Sprintf("%s:score:%s", zsetKey, memberStr)
		ca.Delete(ctx, scoreKey) // Ignore errors
	}

	return nil
}

// Expire sets expiration for a key
func (ca *CacheAdapter) Expire(ctx context.Context, key string, expiration time.Duration) error {
	// Get current value and reset with new expiration
	var value interface{}
	err := ca.manager.Get(ctx, key, &value)
	if err != nil {
		return err
	}

	return ca.manager.Set(ctx, key, value, expiration)
}

// TTL returns time to live for a key
func (ca *CacheAdapter) TTL(ctx context.Context, key string) (time.Duration, error) {
	// This is a limitation of the current implementation
	// In a real Redis implementation, TTL would be tracked separately
	return time.Hour, nil // Default TTL
}

// Flush clears all cache
func (ca *CacheAdapter) Flush(ctx context.Context) error {
	// Clear memory cache
	if ca.manager.memory != nil {
		ca.manager.memory.Clear()
	}

	// For Redis and database, we would need specific flush methods
	return nil
}

// Close closes the cache
func (ca *CacheAdapter) Close() error {
	return ca.manager.Close()
}

// SetFundingRate sets funding rate for a symbol
func (ca *CacheAdapter) SetFundingRate(ctx context.Context, symbol string, rate interface{}, expiration time.Duration) error {
	key := fmt.Sprintf("funding_rate:%s", symbol)
	return ca.Set(ctx, key, rate, expiration)
}

// GetFundingRate gets funding rate for a symbol
func (ca *CacheAdapter) GetFundingRate(ctx context.Context, symbol string, dest interface{}) error {
	key := fmt.Sprintf("funding_rate:%s", symbol)
	return ca.Get(ctx, key, dest)
}

// SetIndexPrice sets index price for a symbol
func (ca *CacheAdapter) SetIndexPrice(ctx context.Context, symbol string, price interface{}, expiration time.Duration) error {
	key := fmt.Sprintf("index_price:%s", symbol)
	return ca.Set(ctx, key, price, expiration)
}

// GetIndexPrice gets index price for a symbol
func (ca *CacheAdapter) GetIndexPrice(ctx context.Context, symbol string, dest interface{}) error {
	key := fmt.Sprintf("index_price:%s", symbol)
	return ca.Get(ctx, key, dest)
}

// CheckRateLimit checks rate limit for a key
func (ca *CacheAdapter) CheckRateLimit(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	rateLimitKey := fmt.Sprintf("rate_limit:%s", key)
	
	// Get current count
	var count int
	err := ca.Get(ctx, rateLimitKey, &count)
	if err != nil {
		// Key doesn't exist, start counting
		count = 0
	}

	if count >= limit {
		return false, nil // Rate limit exceeded
	}

	// Increment count
	count++
	if err := ca.Set(ctx, rateLimitKey, count, window); err != nil {
		return false, err
	}

	return true, nil // Rate limit OK
}

// SetOrderBook sets order book for a symbol
func (ca *CacheAdapter) SetOrderBook(ctx context.Context, symbol string, snapshot interface{}, expiration time.Duration) error {
	key := fmt.Sprintf("order_book:%s", symbol)
	return ca.Set(ctx, key, snapshot, expiration)
}

// GetOrderBook gets order book for a symbol
func (ca *CacheAdapter) GetOrderBook(ctx context.Context, symbol string, dest interface{}) error {
	key := fmt.Sprintf("order_book:%s", symbol)
	return ca.Get(ctx, key, dest)
}

// Helper methods

func (ca *CacheAdapter) getList(ctx context.Context, key string) ([]interface{}, error) {
	var value interface{}
	err := ca.manager.Get(ctx, key, &value)
	if err != nil {
		return []interface{}{}, err
	}

	if list, ok := value.([]interface{}); ok {
		return list, nil
	}

	return []interface{}{}, fmt.Errorf("value is not a list")
}

func (ca *CacheAdapter) getSet(ctx context.Context, key string) (map[string]bool, error) {
	var value interface{}
	err := ca.manager.Get(ctx, key, &value)
	if err != nil {
		return make(map[string]bool), err
	}

	if set, ok := value.(map[string]bool); ok {
		return set, nil
	}

	return make(map[string]bool), fmt.Errorf("value is not a set")
}

func (ca *CacheAdapter) convertValue(value interface{}, dest interface{}) error {
	// If dest is a pointer to interface{}, just assign the value
	if destPtr, ok := dest.(*interface{}); ok {
		*destPtr = value
		return nil
	}

	// Try to convert based on dest type
	switch destPtr := dest.(type) {
	case *string:
		*destPtr = fmt.Sprintf("%v", value)
		return nil
	case *int:
		if intVal, ok := value.(int); ok {
			*destPtr = intVal
			return nil
		}
		if strVal, ok := value.(string); ok {
			intVal, err := strconv.Atoi(strVal)
			if err != nil {
				return err
			}
			*destPtr = intVal
			return nil
		}
		return fmt.Errorf("cannot convert %T to int", value)
	case *float64:
		if floatVal, ok := value.(float64); ok {
			*destPtr = floatVal
			return nil
		}
		if strVal, ok := value.(string); ok {
			floatVal, err := strconv.ParseFloat(strVal, 64)
			if err != nil {
				return err
			}
			*destPtr = floatVal
			return nil
		}
		return fmt.Errorf("cannot convert %T to float64", value)
	default:
		// Try JSON marshaling/unmarshaling
		valueBytes, err := json.Marshal(value)
		if err != nil {
			return fmt.Errorf("failed to marshal value: %w", err)
		}
		return json.Unmarshal(valueBytes, dest)
	}
}