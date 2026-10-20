import { useState, useEffect, useRef } from 'react';
import type { KeyboardEvent } from 'react';
import { isAxiosError } from 'axios';
import { chatApi } from '../api';

interface Message { id?: number; role: string; content: string }
interface Conversation { id: number; title: string }
interface Pending { requestId: string; content: string }
const pendingKey = (id: number) => `hr-chat-pending:${id}`;

function errorMessage(error: unknown): string {
  return isAxiosError(error)
    ? error.response?.data?.message || (error.code === 'ERR_CANCELED' ? '请求已停止，可重试确认结果。' : '请求未完成，请重试确认结果。')
    : '操作失败，请稍后重试。';
}

function readPending(id: number): Pending | null {
  try {
    const value = JSON.parse(sessionStorage.getItem(pendingKey(id)) || 'null');
    return value && typeof value.requestId === 'string' && typeof value.content === 'string' ? value : null;
  } catch { return null; }
}

export default function Chat() {
  const [conversations, setConversations] = useState<Conversation[]>([]);
  const [conversationCursor, setConversationCursor] = useState(0);
  const [activeId, setActiveId] = useState(0);
  const [newTitle, setNewTitle] = useState('');
  const [messages, setMessages] = useState<Message[]>([]);
  const [historyCursor, setHistoryCursor] = useState(0);
  const [input, setInput] = useState('');
  const [pending, setPending] = useState<Pending | null>(null);
  const [loading, setLoading] = useState(false);
  const [fetching, setFetching] = useState(true);
  const [ready, setReady] = useState(false);
  const [error, setError] = useState('');
  const [reload, setReload] = useState(0);
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const controller = useRef<AbortController | null>(null);
  const busy = useRef(false);

  useEffect(() => {
    const abort = new AbortController();
    chatApi.conversations(undefined, abort.signal).then((res) => {
      const items: Conversation[] = res.data.data.conversations || [];
      setConversations(items);
      setConversationCursor(res.data.data.nextBeforeId || 0);
      setActiveId(items[0]?.id || 0);
      if (!items.length) setFetching(false);
    }).catch((err: unknown) => {
      if (!abort.signal.aborted) { setError(errorMessage(err)); setFetching(false); }
    });
    return () => { abort.abort(); controller.current?.abort(); };
  }, []);

  useEffect(() => {
    if (!activeId) return;
    const abort = new AbortController();
    chatApi.history(activeId, undefined, abort.signal).then((res) => {
      setMessages(res.data.data.messages || []);
      setHistoryCursor(res.data.data.nextBeforeId || 0);
      setPending(readPending(activeId));
      setReady(true);
    }).catch((err: unknown) => {
      if (!abort.signal.aborted) setError(errorMessage(err));
    }).finally(() => { if (!abort.signal.aborted) setFetching(false); });
    return () => abort.abort();
  }, [activeId, reload]);

  useEffect(() => { messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' }); }, [activeId, pending, loading]);

  function openConversation(id: number) {
    if (busy.current) return;
    setMessages([]); setPending(null); setInput(''); setError(''); setReady(false);
    setFetching(id !== 0); setHistoryCursor(0); setActiveId(id); setReload((v) => v + 1);
  }

  async function createConversation() {
    if (busy.current) return;
    busy.current = true; setFetching(true); setError('');
    try {
      const res = await chatApi.createConversation(newTitle.trim());
      const item: Conversation = res.data.data;
      setConversations((prev) => [item, ...prev]); setNewTitle('');
      busy.current = false; openConversation(item.id);
    } catch (err) { setError(errorMessage(err)); setFetching(false); }
    finally { busy.current = false; }
  }

  async function deleteConversation() {
    if (busy.current || !activeId || !window.confirm('删除此对话及其全部记录？')) return;
    busy.current = true; setFetching(true); setError('');
    try {
      await chatApi.deleteConversation(activeId);
      sessionStorage.removeItem(pendingKey(activeId));
      const remaining = conversations.filter((c) => c.id !== activeId);
      setConversations(remaining); busy.current = false; openConversation(remaining[0]?.id || 0);
    } catch (err) { setError(errorMessage(err)); setFetching(false); }
    finally { busy.current = false; }
  }

  async function loadMoreConversations() {
    if (busy.current) return;
    busy.current = true; setFetching(true); setError('');
    try {
      const res = await chatApi.conversations(conversationCursor);
      setConversations((prev) => [...prev, ...(res.data.data.conversations || [])]);
      setConversationCursor(res.data.data.nextBeforeId || 0);
    } catch (err) { setError(errorMessage(err)); }
    finally { busy.current = false; setFetching(false); }
  }

  async function loadOlder() {
    if (busy.current) return;
    busy.current = true; setFetching(true); setError('');
    try {
      const res = await chatApi.history(activeId, historyCursor);
      setMessages((prev) => [...(res.data.data.messages || []), ...prev]);
      setHistoryCursor(res.data.data.nextBeforeId || 0);
    } catch (err) { setError(errorMessage(err)); }
    finally { busy.current = false; setFetching(false); }
  }

  async function send(retry?: Pending) {
    if (busy.current || fetching || !ready || !activeId) return;
    const content = retry?.content || input.trim();
    if (!content || Array.from(content).length > 4000) return;
    const request = retry || {
      content,
      requestId: Array.from(crypto.getRandomValues(new Uint8Array(16)), (v) => v.toString(16).padStart(2, '0')).join(''),
    };
    // Retain the ID across a page refresh or transport timeout.
    try { sessionStorage.setItem(pendingKey(activeId), JSON.stringify(request)); }
    catch { setError('无法保存待发送消息，请检查浏览器存储设置后重试。'); return; }
    busy.current = true; setPending(request); setInput(''); setLoading(true); setError('');
    controller.current = new AbortController();
    try {
      await chatApi.send(activeId, request.content, request.requestId, controller.current.signal);
      sessionStorage.removeItem(pendingKey(activeId));
      setPending(null); setReady(false);
      // The database is authoritative, including a reply recovered by a retry.
      try {
        const res = await chatApi.history(activeId);
        setMessages(res.data.data.messages || []);
        setHistoryCursor(res.data.data.nextBeforeId || 0);
        setReady(true);
      } catch { setError('回复已保存，加载记录失败，请刷新记录。'); }
    } catch (err) { setError(errorMessage(err)); }
    finally { busy.current = false; setLoading(false); controller.current = null; }
  }

  function handleKeyDown(e: KeyboardEvent<HTMLTextAreaElement>) {
    if (e.key === 'Enter' && !e.shiftKey && !e.nativeEvent.isComposing) {
      e.preventDefault(); if (!pending) void send();
    }
  }

  const disabled = loading || fetching;
  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%', gap: 12 }}>
      <div style={{ display: 'flex', flexWrap: 'wrap', gap: 8 }}>
        <select aria-label="当前对话" value={activeId} disabled={disabled} onChange={(e) => openConversation(Number(e.target.value))} style={{ maxWidth: '100%', minWidth: 180 }}>
          {!activeId && <option value={0}>请选择或新建对话</option>}
          {conversations.map((c) => <option key={c.id} value={c.id}>{c.title}（{c.id}）</option>)}
        </select>
        {conversationCursor > 0 && <button disabled={disabled} onClick={loadMoreConversations}>更多对话</button>}
        <input aria-label="新对话名称" placeholder="新对话名称（可选）" maxLength={80} value={newTitle} disabled={disabled} onChange={(e) => setNewTitle(e.target.value)} />
        <button disabled={disabled} onClick={createConversation}>新建对话</button>
        <button disabled={disabled || !activeId} onClick={deleteConversation}>删除对话</button>
        <button disabled={disabled || !activeId} onClick={() => openConversation(activeId)}>刷新记录</button>
      </div>
      {error && <div role="alert" style={{ color: '#b42318' }}>{error}</div>}
      <div aria-busy={disabled} style={{ flex: 1, overflowY: 'auto', padding: 16, background: '#f5f5f5', borderRadius: 8 }}>
        {historyCursor > 0 && <button disabled={disabled} onClick={loadOlder}>加载更早记录</button>}
        {!messages.length && !fetching && <div style={{ textAlign: 'center', color: '#666', marginTop: 40 }}>
          <p>{activeId ? '你好，我是招聘助手 AI。可以查询你负责的岗位和候选人。' : '新建对话后开始提问，不同对话的上下文相互独立。'}</p>
        </div>}
        {messages.map((msg, idx) => <div key={msg.id || `reply-${idx}`} style={{ display: 'flex', justifyContent: msg.role === 'user' ? 'flex-end' : 'flex-start', marginBottom: 12 }}>
          <div style={{ maxWidth: '85%', padding: '10px 16px', borderRadius: 12, background: msg.role === 'user' ? '#1677ff' : '#fff', color: msg.role === 'user' ? '#fff' : '#333', whiteSpace: 'pre-wrap', wordBreak: 'break-word' }}>{msg.content}</div>
        </div>)}
        {pending && <div style={{ padding: 12, background: '#fff', border: '1px solid #d9d9d9', borderRadius: 8, whiteSpace: 'pre-wrap', wordBreak: 'break-word' }}>
          <div>{pending.content}</div>
          <p role="status">{loading ? 'AI 思考中…' : '此问题尚未确认完成，请重试以确认结果。'}</p>
          {loading ? <button onClick={() => controller.current?.abort()}>停止生成</button> : <>
            <button disabled={fetching || !ready} onClick={() => send(pending)}>重试此问题</button>
            <button disabled={fetching} onClick={() => { sessionStorage.removeItem(pendingKey(activeId)); setPending(null); setError(''); }}>放弃重试</button>
          </>}
        </div>}
        <div ref={messagesEndRef} />
      </div>
      <div style={{ display: 'flex', gap: 8 }}>
        <textarea aria-label="你的问题" value={input} onChange={(e) => setInput(e.target.value)} onKeyDown={handleKeyDown} disabled={disabled || !activeId || !ready || !!pending} placeholder="输入你的问题，最多 4000 字…" style={{ flex: 1, padding: 12, borderRadius: 8, border: '1px solid #d9d9d9', resize: 'vertical', minHeight: 48, fontFamily: 'inherit', fontSize: 14 }} />
        <button onClick={() => send()} disabled={disabled || !activeId || !ready || !!pending || !input.trim() || Array.from(input.trim()).length > 4000} style={{ padding: '0 24px', background: '#1677ff', color: '#fff', border: 'none', borderRadius: 8, cursor: 'pointer', fontSize: 14 }}>发送</button>
      </div>
    </div>
  );
}
