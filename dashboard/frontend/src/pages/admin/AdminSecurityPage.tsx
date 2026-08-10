import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Button, Form, Input, Typography, message } from 'antd'
import { useTranslation } from 'react-i18next'
import { api } from '../../api'
import styles from '../Portal.module.scss'

export function AdminSecurityPage() {
  const { t } = useTranslation()
  const client = useQueryClient()
  const me = useQuery({ queryKey: ['admin-me'], queryFn: api.adminMe })
  const username = useMutation({ mutationFn: (v: { username: string }) => api.updateAccount(v.username), onSuccess: () => { void client.invalidateQueries({ queryKey: ['admin-me'] }); void message.success(t('saved')) } })
  const password = useMutation({ mutationFn: (v: { current_password: string; new_password: string }) => api.updatePassword(v.current_password, v.new_password), onSuccess: () => void message.success(t('passwordUpdated')) })

  return <div className={styles.adminPage}>
    <div className={styles.pageHeader}><Typography.Title level={2}>{t('account')}</Typography.Title></div>
    <div className={styles.accountGrid}>
      <section className={styles.accountColumn}>
        <Form layout="vertical" initialValues={{ username: me.data?.username }} onFinish={v => username.mutate(v)}>
          <Form.Item name="username" label={t('username')} rules={[{ required: true, message: t('usernameRequired') }, { min: 3, max: 64, message: t('usernameLength') }]}><Input /></Form.Item>
          <Button htmlType="submit" type="primary" loading={username.isPending}>{t('save')}</Button>
        </Form>
      </section>
      <section className={styles.accountColumn}>
        <Form layout="vertical" onFinish={v => password.mutate(v)}>
          <Form.Item name="current_password" label={t('currentPassword')} rules={[{ required: true, message: t('passwordRequired') }]}><Input.Password autoComplete="current-password" /></Form.Item>
          <Form.Item name="new_password" label={t('newPassword')} rules={[{ required: true, message: t('passwordRequired') }, { min: 12, max: 72, message: t('passwordLength') }]}><Input.Password autoComplete="new-password" /></Form.Item>
          <Form.Item name="confirm_password" label={t('confirmPassword')} dependencies={['new_password']} rules={[{ required: true, message: t('confirmPasswordRequired') }, ({ getFieldValue }) => ({ validator: (_, value) => !value || getFieldValue('new_password') === value ? Promise.resolve() : Promise.reject(new Error(t('passwordMismatch'))) })]}><Input.Password autoComplete="new-password" /></Form.Item>
          <Button htmlType="submit" loading={password.isPending}>{t('update')}</Button>
        </Form>
      </section>
    </div>
  </div>
}
