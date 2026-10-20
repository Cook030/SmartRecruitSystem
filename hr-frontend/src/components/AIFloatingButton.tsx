import { useState, useRef, useEffect, useCallback } from 'react';
import Chat from '../pages/Chat';

export default function AIFloatingButton() {
  const [isOpen, setIsOpen] = useState(false);
  const [position, setPosition] = useState({ x: window.innerWidth - 84, y: window.innerHeight - 84 });
  const [isDragging, setIsDragging] = useState(false);
  const dragOffset = useRef({ x: 0, y: 0 });
  const buttonRef = useRef<HTMLDivElement>(null);
  const hasMoved = useRef(false);
  const startPos = useRef({ x: 0, y: 0 });

  const handleMouseDown = (e: React.MouseEvent) => {
    setIsDragging(true);
    hasMoved.current = false;
    startPos.current = { x: e.clientX, y: e.clientY };
    const rect = buttonRef.current?.getBoundingClientRect();
    if (rect) {
      dragOffset.current = {
        x: e.clientX - rect.left,
        y: e.clientY - rect.top,
      };
    }
  };

  const handleMouseMove = useCallback((e: MouseEvent) => {
    const deltaX = Math.abs(e.clientX - startPos.current.x);
    const deltaY = Math.abs(e.clientY - startPos.current.y);
    if (deltaX > 3 || deltaY > 3) {
      hasMoved.current = true;
    }
    const newX = e.clientX - dragOffset.current.x;
    const newY = e.clientY - dragOffset.current.y;
    setPosition({
      x: Math.max(0, Math.min(window.innerWidth - 60, newX)),
      y: Math.max(0, Math.min(window.innerHeight - 60, newY)),
    });
  }, []);

  const handleMouseUp = useCallback(() => {
    if (!hasMoved.current) {
      setIsOpen(true);
    }
    setIsDragging(false);
    hasMoved.current = false;
  }, []);

  useEffect(() => {
    if (!isDragging) return;
    window.addEventListener('mousemove', handleMouseMove);
    window.addEventListener('mouseup', handleMouseUp);
    return () => {
      window.removeEventListener('mousemove', handleMouseMove);
      window.removeEventListener('mouseup', handleMouseUp);
    };
  }, [isDragging, handleMouseMove, handleMouseUp]);

  return (
    <>
      {/* 悬浮球按钮 */}
      <div
        ref={buttonRef}
        onMouseDown={handleMouseDown}
        style={{
          position: 'fixed',
          left: position.x,
          top: position.y,
          width: 60,
          height: 60,
          borderRadius: '50%',
          background: '#fff',
          boxShadow: '0 4px 12px rgba(102, 126, 234, 0.4)',
          cursor: isDragging ? 'grabbing' : 'grab',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          zIndex: 999,
          transition: isDragging ? 'none' : 'all 0.2s ease',
          userSelect: 'none',
        }}
        onMouseOver={(e) => {
          if (!isDragging) {
            e.currentTarget.style.transform = 'scale(1.1)';
            e.currentTarget.style.boxShadow = '0 6px 16px rgba(102, 126, 234, 0.6)';
          }
        }}
        onMouseOut={(e) => {
          if (!isDragging) {
            e.currentTarget.style.transform = 'scale(1)';
            e.currentTarget.style.boxShadow = '0 4px 12px rgba(102, 126, 234, 0.4)';
          }
        }}
      >
        <span style={{ fontSize: 28 }}>🤖</span>
      </div>

      {/* AI 对话弹窗 */}
      {isOpen && (
        <ResizableWindow onClose={() => setIsOpen(false)}>
          <Chat />
        </ResizableWindow>
      )}
    </>
  );
}

// 可调节大小的窗口组件
function ResizableWindow({ children, onClose }: { children: React.ReactNode; onClose: () => void }) {
  const [size, setSize] = useState({ width: 800, height: 600 });
  const [isResizing, setIsResizing] = useState(false);
  const resizeStartPos = useRef({ x: 0, y: 0, width: 0, height: 0 });

  const handleResizeStart = (e: React.MouseEvent) => {
    e.preventDefault();
    setIsResizing(true);
    resizeStartPos.current = {
      x: e.clientX,
      y: e.clientY,
      width: size.width,
      height: size.height,
    };
  };

  const handleResizeMove = useCallback((e: MouseEvent) => {
    if (!isResizing) return;
    const deltaX = e.clientX - resizeStartPos.current.x;
    const deltaY = e.clientY - resizeStartPos.current.y;
    setSize({
      width: Math.max(400, Math.min(window.innerWidth * 0.9, resizeStartPos.current.width + deltaX)),
      height: Math.max(400, Math.min(window.innerHeight * 0.8, resizeStartPos.current.height + deltaY)),
    });
  }, [isResizing]);

  const handleResizeEnd = useCallback(() => {
    setIsResizing(false);
  }, []);

  useEffect(() => {
    if (isResizing) {
      window.addEventListener('mousemove', handleResizeMove);
      window.addEventListener('mouseup', handleResizeEnd);
      return () => {
        window.removeEventListener('mousemove', handleResizeMove);
        window.removeEventListener('mouseup', handleResizeEnd);
      };
    }
  }, [isResizing, handleResizeMove, handleResizeEnd]);

  return (
    <div
      style={{
        position: 'fixed',
        top: 0,
        left: 0,
        right: 0,
        bottom: 0,
        background: 'rgba(0, 0, 0, 0.5)',
        zIndex: 1000,
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
      }}
      onClick={onClose}
    >
      <div
        onClick={(e) => e.stopPropagation()}
        style={{
          width: size.width,
          height: size.height,
          background: '#fff',
          borderRadius: 12,
          boxShadow: '0 8px 32px rgba(0, 0, 0, 0.2)',
          display: 'flex',
          flexDirection: 'column',
          position: 'relative',
          minWidth: 400,
          minHeight: 400,
        }}
      >
        {/* 窗口标题栏 */}
        <div
          style={{
            height: 40,
            background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
            borderRadius: '12px 12px 0 0',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            padding: '0 16px',
            cursor: 'move',
            userSelect: 'none',
          }}
        >
          <span style={{ color: '#fff', fontWeight: 'bold', fontSize: 14 }}>🤖 AI 智能助手</span>
          <button
            onClick={onClose}
            style={{
              width: 28,
              height: 28,
              borderRadius: '50%',
              background: 'rgba(255, 255, 255, 0.2)',
              border: 'none',
              cursor: 'pointer',
              fontSize: 20,
              color: '#fff',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              transition: 'all 0.2s',
            }}
            onMouseOver={(e) => {
              e.currentTarget.style.background = 'rgba(255, 255, 255, 0.3)';
            }}
            onMouseOut={(e) => {
              e.currentTarget.style.background = 'rgba(255, 255, 255, 0.2)';
            }}
          >
            ×
          </button>
        </div>

        {/* 内容区域 */}
        <div style={{ flex: 1, padding: 16, overflow: 'hidden' }}>
          {children}
        </div>

        {/* 右下角调节手柄 */}
        <div
          onMouseDown={handleResizeStart}
          style={{
            position: 'absolute',
            right: 0,
            bottom: 0,
            width: 20,
            height: 20,
            cursor: 'se-resize',
            background: 'linear-gradient(135deg, transparent 50%, #667eea 50%)',
            opacity: isResizing ? 1 : 0.5,
            transition: 'opacity 0.2s',
            borderRadius: '0 0 12px 0',
          }}
          onMouseOver={(e) => {
            e.currentTarget.style.opacity = '1';
          }}
          onMouseOut={(e) => {
            if (!isResizing) {
              e.currentTarget.style.opacity = '0.5';
            }
          }}
        />
      </div>

    </div>
  );
}
