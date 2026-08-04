import { useMutation, useQuery } from '@tanstack/react-query'
import { Alert, Button, Form, Input, Layout, Typography, message } from 'antd'
import { Navigate, useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { api, resetCSRF } from '../api'
import { FloatingNav } from '../components/FloatingNav'
import styles from './Portal.module.scss'

export function AdminLoginPage(){const{t}=useTranslation();const navigate=useNavigate();const status=useQuery({queryKey:['setup-status'],queryFn:api.setupStatus,retry:false});const login=useMutation({mutationFn:(v:{username:string;password:string})=>api.login(v.username,v.password),onSuccess:()=>{resetCSRF();void message.success(t('loginDone'));navigate('/admin')}})
  if(status.data?.required)return <Navigate to="/admin/setup" replace/>
  return <Layout className={styles.layout}><FloatingNav mode="auth"/><main className={styles.auth}><section className={styles.authPanel}><div className={styles.authHeading}><Typography.Title level={2}>{t('adminLogin')}</Typography.Title></div>{login.isError&&<Alert type="error" showIcon title={t('loginFailed')} style={{marginBottom:16}}/>}<Form layout="vertical" onFinish={v=>login.mutate(v)}><Form.Item name="username" label={t('username')} rules={[{required:true,message:t('usernameRequired')}]}><Input autoComplete="username"/></Form.Item><Form.Item name="password" label={t('password')} rules={[{required:true,message:t('passwordRequired')}]}><Input.Password autoComplete="current-password"/></Form.Item><Button type="primary" htmlType="submit" block loading={login.isPending}>{t('login')}</Button></Form></section></main></Layout>}
