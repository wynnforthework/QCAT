"use client";

import React, { createContext, useContext, useState, useEffect, ReactNode } from 'react';
import apiClient, { AuthResponse, LoginRequest } from '@/lib/api';

interface User {
  id: string;
  username: string;
  role: string;
}

interface AuthContextType {
  user: User | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  login: (credentials: LoginRequest) => Promise<void>;
  logout: () => void;
  refreshAuth: () => Promise<void>;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export const useAuth = () => {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
};

interface AuthProviderProps {
  children: ReactNode;
}

export const AuthProvider: React.FC<AuthProviderProps> = ({ children }) => {
  const [user, setUser] = useState<User | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  const isAuthenticated = !!user;

  // 保存认证信息到localStorage
  const saveAuthData = (authData: AuthResponse) => {
    localStorage.setItem('accessToken', authData.access_token);
    localStorage.setItem('refreshToken', authData.refresh_token);
    localStorage.setItem('tokenExpiresAt', authData.expires_at);

    setUser({
      id: authData.user_id,
      username: authData.username,
      role: authData.role,
    });
  };

  // 清除认证信息
  const clearAuthData = () => {
    localStorage.removeItem('accessToken');
    localStorage.removeItem('refreshToken');
    localStorage.removeItem('tokenExpiresAt');
    setUser(null);
  };

  // 检查token是否过期
  const isTokenExpired = (): boolean => {
    const expiresAt = localStorage.getItem('tokenExpiresAt');
    if (!expiresAt) return true;
    
    return new Date(expiresAt) <= new Date();
  };

  // 刷新token
  const refreshAuth = async (): Promise<void> => {
    try {
      const refreshToken = localStorage.getItem('refreshToken');
      if (!refreshToken) {
        throw new Error('No refresh token available');
      }

      const authData = await apiClient.refreshToken({ refresh_token: refreshToken });
      saveAuthData(authData);
    } catch (error) {
      console.error('Failed to refresh token:', error);
      clearAuthData();
      throw error;
    }
  };

  // 登录
  const login = async (credentials: LoginRequest): Promise<void> => {
    try {
      setIsLoading(true);
      const authData = await apiClient.login(credentials);
      saveAuthData(authData);
    } catch (error) {
      console.error('Login failed:', error);
      throw error;
    } finally {
      setIsLoading(false);
    }
  };

  // 登出
  const logout = () => {
    clearAuthData();
  };

  // 初始化认证状态
  useEffect(() => {
    const initAuth = async () => {
      try {
        const accessToken = localStorage.getItem('accessToken');
        const refreshToken = localStorage.getItem('refreshToken');
        
        if (!accessToken || !refreshToken) {
          setIsLoading(false);
          return;
        }

        // 检查token是否过期
        if (isTokenExpired()) {
          try {
            await refreshAuth();
          } catch (error) {
            console.error('Failed to refresh token on init:', error);
            clearAuthData();
          }
        } else {
          // Token还有效，从localStorage恢复用户信息
          // 这里我们需要从token中解析用户信息，或者调用一个获取用户信息的接口
          // 暂时先设置一个占位符，后续可以优化
          const userInfo = {
            id: 'temp-id',
            username: 'temp-username',
            role: 'user'
          };
          setUser(userInfo);
        }
      } catch (error) {
        console.error('Auth initialization failed:', error);
        clearAuthData();
      } finally {
        setIsLoading(false);
      }
    };

    initAuth();
  }, []);

  const value: AuthContextType = {
    user,
    isAuthenticated,
    isLoading,
    login,
    logout,
    refreshAuth,
  };

  return (
    <AuthContext.Provider value={value}>
      {children}
    </AuthContext.Provider>
  );
};
