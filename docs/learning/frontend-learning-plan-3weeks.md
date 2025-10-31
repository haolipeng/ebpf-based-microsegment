# 3周前端实战速成计划

**适用人群**: 有C/C++/Go后端经验的程序员
**AI工具**: Claude Code / Cursor / GitHub Copilot
**目标**: 3周后能独立开发全栈Web应用
**学习时间**: 每天2-3小时，周末4-6小时

---

## 📅 Week 1: 基础速成 + 第一个项目

### 🎯 本周目标
- ✅ 掌握 TypeScript 核心语法
- ✅ 理解 React 基本概念
- ✅ 完成一个可运行的 Todo App
- ✅ 部署到 Vercel

---

### 📅 Day 1 (Monday): 环境搭建 + TypeScript入门

#### 上午 (1.5小时): 开发环境准备

**任务清单**:
```bash
# 1. 安装 Node.js LTS
https://nodejs.org/  # 下载 v20.x

# 2. 验证安装
node --version  # v20.x.x
npm --version   # 10.x.x

# 3. 安装 VSCode 插件
- ES7+ React/Redux/React-Native snippets
- Tailwind CSS IntelliSense
- Prettier - Code formatter
- ESLint
- Error Lens

# 4. 创建第一个项目
npx create-next-app@latest week1-learning --typescript --tailwind --app --use-npm

cd week1-learning
npm run dev  # 打开 http://localhost:3000
```

**✅ 完成标准**: 能看到 Next.js 默认首页

#### 下午 (1.5小时): TypeScript 核心语法

**学习资料**: [TypeScript 5分钟快速上手](https://www.typescriptlang.org/docs/handbook/typescript-in-5-minutes.html)

**必学知识点**:
```typescript
// 1. 基础类型
let name: string = "Alice";
let age: number = 30;
let isActive: boolean = true;
let numbers: number[] = [1, 2, 3];

// 2. 接口和类型
interface User {
    id: number;
    name: string;
    email?: string;  // 可选
}

type Status = "active" | "inactive" | "pending";

// 3. 函数类型
function add(a: number, b: number): number {
    return a + b;
}

const multiply = (a: number, b: number): number => a * b;

// 4. 泛型（重要！）
function first<T>(arr: T[]): T | undefined {
    return arr[0];
}

// 5. 联合类型和交叉类型
type ID = string | number;
type Admin = User & { role: "admin" };

// 6. 实用工具类型
type PartialUser = Partial<User>;     // 所有属性可选
type RequiredUser = Required<User>;   // 所有属性必需
type ReadonlyUser = Readonly<User>;   // 所有属性只读
type UserName = Pick<User, "name">;   // 选择属性
```

**实战练习**:
```typescript
// 创建 src/types/todo.ts
export interface Todo {
    id: number;
    title: string;
    completed: boolean;
    createdAt: Date;
}

export type TodoInput = Omit<Todo, "id" | "createdAt">;

// 练习：定义一个通用的API响应类型
export interface ApiResponse<T> {
    success: boolean;
    data?: T;
    error?: string;
}
```

**📚 学习资源**:
- 视频: [TypeScript Crash Course](https://www.youtube.com/watch?v=BCg4U1FzODs) (B站搜中文版)
- 文档: https://www.typescriptlang.org/docs/handbook/

**✅ 完成标准**:
- [ ] 理解接口和类型别名的区别
- [ ] 能定义函数的参数和返回值类型
- [ ] 理解泛型的基本用法

---

### 📅 Day 2 (Tuesday): JavaScript 核心特性

#### 上午 (1.5小时): 现代 JavaScript 语法

**必学知识点**:
```javascript
// 1. 解构赋值（重要！React中大量使用）
const user = { name: "Bob", age: 30, email: "bob@example.com" };
const { name, age } = user;

const arr = [1, 2, 3, 4, 5];
const [first, second, ...rest] = arr;  // rest = [3, 4, 5]

// 2. 展开运算符（重要！）
const newUser = { ...user, age: 31 };  // 创建新对象
const newArr = [...arr, 6, 7];         // 创建新数组

// 3. 箭头函数
const add = (a, b) => a + b;
const square = x => x * x;  // 单参数可省略括号

// 4. 数组方法（重要！React中大量使用）
const numbers = [1, 2, 3, 4, 5];

const doubled = numbers.map(n => n * 2);           // [2, 4, 6, 8, 10]
const evens = numbers.filter(n => n % 2 === 0);    // [2, 4]
const sum = numbers.reduce((acc, n) => acc + n, 0); // 15
const found = numbers.find(n => n > 3);            // 4

// 5. 可选链和空值合并（重要！）
const value = user?.address?.city ?? "Unknown";
const port = process.env.PORT ?? 3000;

// 6. 模板字符串
const greeting = `Hello, ${name}! You are ${age} years old.`;

// 7. Promise 和 async/await
async function fetchUser(id: number): Promise<User> {
    const response = await fetch(`/api/users/${id}`);
    if (!response.ok) throw new Error("Failed to fetch");
    return response.json();
}

// 8. try/catch 错误处理
async function safeRequest() {
    try {
        const data = await fetchUser(1);
        console.log(data);
    } catch (error) {
        console.error("Error:", error);
    }
}
```

**实战练习**:
```javascript
// 练习：处理数组数据
const todos = [
    { id: 1, title: "Learn React", completed: false },
    { id: 2, title: "Build app", completed: true },
    { id: 3, title: "Deploy", completed: false }
];

// 1. 获取所有未完成的todo
const pending = todos.filter(todo => !todo.completed);

// 2. 标记ID为2的todo为完成
const updated = todos.map(todo =>
    todo.id === 2 ? { ...todo, completed: true } : todo
);

// 3. 计算完成率
const completionRate = todos.reduce((acc, todo) =>
    acc + (todo.completed ? 1 : 0), 0
) / todos.length * 100;
```

#### 下午 (1.5小时): 模块系统和包管理

**学习内容**:
```javascript
// ES6 模块
// math.ts
export const add = (a: number, b: number) => a + b;
export const PI = 3.14159;

// app.ts
import { add, PI } from './math';
import * as math from './math';  // 导入所有

// 默认导出
// Button.tsx
export default function Button() { return <button>Click</button>; }

// App.tsx
import Button from './Button';
```

**NPM 常用命令**:
```bash
# 安装依赖
npm install <package-name>
npm install -D <package-name>  # 开发依赖

# 常用包
npm install axios              # HTTP客户端
npm install date-fns           # 日期处理
npm install zod                # 运行时类型验证
npm install clsx               # className工具

# 脚本命令
npm run dev        # 开发服务器
npm run build      # 生产构建
npm run lint       # 代码检查
```

**✅ 完成标准**:
- [ ] 理解解构和展开运算符
- [ ] 熟练使用数组方法（map, filter, reduce）
- [ ] 理解async/await异步处理
- [ ] 能导入导出模块

---

### 📅 Day 3 (Wednesday): React 核心概念

#### 全天 (3小时): React 基础

**学习资料**: [React 官方教程](https://react.dev/learn)

**核心概念 1: 组件**
```tsx
// app/components/Greeting.tsx

// 函数组件（就是一个返回JSX的函数）
function Greeting() {
    return <h1>Hello, World!</h1>;
}

// 带Props的组件
interface GreetingProps {
    name: string;
    age?: number;
}

function Greeting({ name, age }: GreetingProps) {
    return (
        <div>
            <h1>Hello, {name}!</h1>
            {age && <p>You are {age} years old.</p>}
        </div>
    );
}

// 使用组件
<Greeting name="Alice" age={30} />
```

**核心概念 2: JSX**
```tsx
// JSX 就是 JavaScript + XML
function TodoItem({ todo }: { todo: Todo }) {
    return (
        <div className="todo-item">
            {/* 1. 大括号内是JS表达式 */}
            <h3>{todo.title}</h3>

            {/* 2. 条件渲染 */}
            {todo.completed && <span>✓</span>}

            {/* 3. 三元运算符 */}
            <span>{todo.completed ? "Done" : "Pending"}</span>

            {/* 4. 列表渲染 */}
            <ul>
                {items.map(item => (
                    <li key={item.id}>{item.name}</li>
                ))}
            </ul>

            {/* 5. 事件处理 */}
            <button onClick={() => handleClick(todo.id)}>
                Delete
            </button>
        </div>
    );
}
```

**核心概念 3: useState（状态管理）**
```tsx
import { useState } from 'react';

function Counter() {
    // useState 返回 [状态值, 更新函数]
    const [count, setCount] = useState(0);

    // 直接更新
    const increment = () => setCount(count + 1);

    // 函数式更新（推荐，避免闭包问题）
    const incrementSafe = () => setCount(prev => prev + 1);

    return (
        <div>
            <p>Count: {count}</p>
            <button onClick={increment}>+1</button>
            <button onClick={() => setCount(0)}>Reset</button>
        </div>
    );
}

// 复杂状态
function TodoList() {
    const [todos, setTodos] = useState<Todo[]>([]);

    const addTodo = (title: string) => {
        const newTodo: Todo = {
            id: Date.now(),
            title,
            completed: false,
            createdAt: new Date()
        };
        setTodos(prev => [...prev, newTodo]);
    };

    const toggleTodo = (id: number) => {
        setTodos(prev => prev.map(todo =>
            todo.id === id ? { ...todo, completed: !todo.completed } : todo
        ));
    };

    const deleteTodo = (id: number) => {
        setTodos(prev => prev.filter(todo => todo.id !== id));
    };
}
```

**核心概念 4: useEffect（副作用）**
```tsx
import { useEffect, useState } from 'react';

function DataFetcher() {
    const [data, setData] = useState(null);
    const [loading, setLoading] = useState(true);

    // 组件挂载时执行（空依赖数组）
    useEffect(() => {
        fetch('/api/data')
            .then(res => res.json())
            .then(data => {
                setData(data);
                setLoading(false);
            });
    }, []);  // [] 表示只运行一次

    // 依赖变化时执行
    useEffect(() => {
        console.log('Data changed:', data);
    }, [data]);  // data变化时运行

    // 清理函数
    useEffect(() => {
        const timer = setInterval(() => {
            console.log('tick');
        }, 1000);

        return () => clearInterval(timer);  // 组件卸载时清理
    }, []);

    if (loading) return <div>Loading...</div>;
    return <div>{JSON.stringify(data)}</div>;
}
```

**实战练习**: 改造 app/page.tsx
```tsx
// app/page.tsx
'use client';

import { useState } from 'react';

export default function Home() {
    const [count, setCount] = useState(0);

    return (
        <main className="flex min-h-screen flex-col items-center justify-center p-24">
            <h1 className="text-4xl font-bold mb-8">Counter App</h1>
            <div className="text-6xl mb-8">{count}</div>
            <div className="flex gap-4">
                <button
                    onClick={() => setCount(prev => prev - 1)}
                    className="px-6 py-3 bg-red-500 text-white rounded-lg hover:bg-red-600"
                >
                    -1
                </button>
                <button
                    onClick={() => setCount(0)}
                    className="px-6 py-3 bg-gray-500 text-white rounded-lg hover:bg-gray-600"
                >
                    Reset
                </button>
                <button
                    onClick={() => setCount(prev => prev + 1)}
                    className="px-6 py-3 bg-green-500 text-white rounded-lg hover:bg-green-600"
                >
                    +1
                </button>
            </div>
        </main>
    );
}
```

**✅ 完成标准**:
- [ ] 理解组件就是函数
- [ ] 会用useState管理状态
- [ ] 理解useEffect的执行时机
- [ ] 能实现简单的交互

---

### 📅 Day 4 (Thursday): Tailwind CSS 速成

#### 全天 (3小时): CSS框架实战

**核心理念**: 不写CSS，用工具类组合

**常用类名速查表**:
```css
/* 布局 */
flex          /* display: flex */
grid          /* display: grid */
block         /* display: block */
hidden        /* display: none */
items-center  /* align-items: center */
justify-between /* justify-content: space-between */

/* 间距 */
p-4           /* padding: 1rem (16px) */
px-4          /* padding-left/right: 1rem */
py-2          /* padding-top/bottom: 0.5rem */
m-4           /* margin: 1rem */
gap-4         /* gap: 1rem */

/* 尺寸 */
w-full        /* width: 100% */
w-1/2         /* width: 50% */
h-screen      /* height: 100vh */
max-w-md      /* max-width: 28rem */

/* 颜色 */
bg-blue-500   /* background-color: 蓝色 */
text-white    /* color: 白色 */
border-gray-300 /* border-color: 灰色 */

/* 文字 */
text-lg       /* font-size: 1.125rem */
text-xl       /* font-size: 1.25rem */
font-bold     /* font-weight: 700 */
text-center   /* text-align: center */

/* 圆角和阴影 */
rounded       /* border-radius: 0.25rem */
rounded-lg    /* border-radius: 0.5rem */
rounded-full  /* border-radius: 9999px */
shadow        /* box-shadow: small */
shadow-lg     /* box-shadow: large */

/* 响应式 */
md:flex       /* @media (min-width: 768px) { display: flex } */
lg:grid       /* @media (min-width: 1024px) { display: grid } */
sm:hidden     /* @media (max-width: 640px) { display: none } */

/* 悬停和交互 */
hover:bg-blue-600    /* :hover { background-color } */
focus:outline-none   /* :focus { outline: none } */
active:scale-95      /* :active { transform: scale(0.95) } */
```

**实战案例**: 卡片组件
```tsx
// components/Card.tsx
interface CardProps {
    title: string;
    description: string;
    imageUrl?: string;
}

export function Card({ title, description, imageUrl }: CardProps) {
    return (
        <div className="max-w-sm rounded-lg overflow-hidden shadow-lg bg-white hover:shadow-xl transition-shadow">
            {imageUrl && (
                <img
                    className="w-full h-48 object-cover"
                    src={imageUrl}
                    alt={title}
                />
            )}
            <div className="px-6 py-4">
                <div className="font-bold text-xl mb-2">{title}</div>
                <p className="text-gray-700 text-base">{description}</p>
            </div>
            <div className="px-6 pt-4 pb-2">
                <span className="inline-block bg-gray-200 rounded-full px-3 py-1 text-sm font-semibold text-gray-700 mr-2 mb-2">
                    #tag
                </span>
            </div>
        </div>
    );
}
```

**响应式设计**:
```tsx
<div className="
    grid
    grid-cols-1       /* 手机: 1列 */
    md:grid-cols-2    /* 平板: 2列 */
    lg:grid-cols-3    /* 桌面: 3列 */
    gap-4
">
    <Card title="Card 1" />
    <Card title="Card 2" />
    <Card title="Card 3" />
</div>
```

**实战练习**: 美化Counter App
```tsx
// app/page.tsx
export default function Home() {
    const [count, setCount] = useState(0);

    return (
        <main className="min-h-screen bg-gradient-to-br from-purple-400 via-pink-500 to-red-500 flex items-center justify-center p-4">
            <div className="bg-white rounded-2xl shadow-2xl p-8 md:p-12 max-w-md w-full">
                <h1 className="text-4xl md:text-5xl font-bold text-center mb-4 bg-gradient-to-r from-purple-600 to-pink-600 bg-clip-text text-transparent">
                    Counter App
                </h1>

                <div className="text-8xl font-bold text-center my-12 text-gray-800">
                    {count}
                </div>

                <div className="grid grid-cols-3 gap-4">
                    <button
                        onClick={() => setCount(prev => prev - 1)}
                        className="px-6 py-4 bg-red-500 text-white rounded-xl font-semibold hover:bg-red-600 active:scale-95 transition-all shadow-lg hover:shadow-xl"
                    >
                        -1
                    </button>
                    <button
                        onClick={() => setCount(0)}
                        className="px-6 py-4 bg-gray-500 text-white rounded-xl font-semibold hover:bg-gray-600 active:scale-95 transition-all shadow-lg hover:shadow-xl"
                    >
                        Reset
                    </button>
                    <button
                        onClick={() => setCount(prev => prev + 1)}
                        className="px-6 py-4 bg-green-500 text-white rounded-xl font-semibold hover:bg-green-600 active:scale-95 transition-all shadow-lg hover:shadow-xl"
                    >
                        +1
                    </button>
                </div>
            </div>
        </main>
    );
}
```

**✅ 完成标准**:
- [ ] 理解 Tailwind 的工具类概念
- [ ] 能快速布局（flex, grid）
- [ ] 会用响应式断点
- [ ] 能做出好看的UI

---

### 📅 Day 5-6 (Fri-Sat): Todo App 项目实战

#### 项目需求
- ✅ 添加、删除、标记完成Todo
- ✅ 数据持久化到localStorage
- ✅ 响应式设计
- ✅ 使用TypeScript
- ✅ 优雅的UI

#### 项目结构
```
app/
├── page.tsx              # 主页面
├── layout.tsx            # 根布局
└── components/
    ├── TodoList.tsx      # Todo列表
    ├── TodoItem.tsx      # 单个Todo
    ├── AddTodo.tsx       # 添加表单
    └── Filter.tsx        # 筛选器
types/
└── todo.ts               # 类型定义
hooks/
└── useTodos.ts           # 自定义Hook
lib/
└── storage.ts            # localStorage工具
```

#### 实现步骤

**Step 1: 类型定义 (types/todo.ts)**
```typescript
export interface Todo {
    id: number;
    title: string;
    completed: boolean;
    createdAt: Date;
}

export type FilterType = 'all' | 'active' | 'completed';
```

**Step 2: localStorage工具 (lib/storage.ts)**
```typescript
const STORAGE_KEY = 'todos';

export const loadTodos = (): Todo[] => {
    if (typeof window === 'undefined') return [];
    const data = localStorage.getItem(STORAGE_KEY);
    if (!data) return [];
    return JSON.parse(data).map((todo: any) => ({
        ...todo,
        createdAt: new Date(todo.createdAt)
    }));
};

export const saveTodos = (todos: Todo[]) => {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(todos));
};
```

**Step 3: 自定义Hook (hooks/useTodos.ts)**
```typescript
'use client';

import { useState, useEffect } from 'react';
import { Todo } from '@/types/todo';
import { loadTodos, saveTodos } from '@/lib/storage';

export function useTodos() {
    const [todos, setTodos] = useState<Todo[]>([]);

    // 加载数据
    useEffect(() => {
        setTodos(loadTodos());
    }, []);

    // 保存数据
    useEffect(() => {
        saveTodos(todos);
    }, [todos]);

    const addTodo = (title: string) => {
        const newTodo: Todo = {
            id: Date.now(),
            title,
            completed: false,
            createdAt: new Date()
        };
        setTodos(prev => [...prev, newTodo]);
    };

    const toggleTodo = (id: number) => {
        setTodos(prev => prev.map(todo =>
            todo.id === id ? { ...todo, completed: !todo.completed } : todo
        ));
    };

    const deleteTodo = (id: number) => {
        setTodos(prev => prev.filter(todo => todo.id !== id));
    };

    return { todos, addTodo, toggleTodo, deleteTodo };
}
```

**Step 4: AddTodo组件 (app/components/AddTodo.tsx)**
```typescript
'use client';

import { useState } from 'react';

interface AddTodoProps {
    onAdd: (title: string) => void;
}

export function AddTodo({ onAdd }: AddTodoProps) {
    const [input, setInput] = useState('');

    const handleSubmit = (e: React.FormEvent) => {
        e.preventDefault();
        if (input.trim()) {
            onAdd(input.trim());
            setInput('');
        }
    };

    return (
        <form onSubmit={handleSubmit} className="flex gap-2">
            <input
                type="text"
                value={input}
                onChange={(e) => setInput(e.target.value)}
                placeholder="What needs to be done?"
                className="flex-1 px-4 py-3 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
            <button
                type="submit"
                className="px-6 py-3 bg-blue-500 text-white rounded-lg font-semibold hover:bg-blue-600 transition-colors"
            >
                Add
            </button>
        </form>
    );
}
```

**Step 5: TodoItem组件 (app/components/TodoItem.tsx)**
```typescript
import { Todo } from '@/types/todo';

interface TodoItemProps {
    todo: Todo;
    onToggle: (id: number) => void;
    onDelete: (id: number) => void;
}

export function TodoItem({ todo, onToggle, onDelete }: TodoItemProps) {
    return (
        <div className="flex items-center gap-3 p-4 bg-white rounded-lg shadow-sm hover:shadow-md transition-shadow">
            <input
                type="checkbox"
                checked={todo.completed}
                onChange={() => onToggle(todo.id)}
                className="w-5 h-5 text-blue-500 rounded focus:ring-2 focus:ring-blue-500"
            />
            <span className={`flex-1 ${todo.completed ? 'line-through text-gray-400' : 'text-gray-800'}`}>
                {todo.title}
            </span>
            <button
                onClick={() => onDelete(todo.id)}
                className="px-3 py-1 text-red-500 hover:bg-red-50 rounded transition-colors"
            >
                Delete
            </button>
        </div>
    );
}
```

**Step 6: 主页面 (app/page.tsx)**
```typescript
'use client';

import { useState } from 'react';
import { useTodos } from '@/hooks/useTodos';
import { AddTodo } from './components/AddTodo';
import { TodoItem } from './components/TodoItem';
import { FilterType } from '@/types/todo';

export default function Home() {
    const { todos, addTodo, toggleTodo, deleteTodo } = useTodos();
    const [filter, setFilter] = useState<FilterType>('all');

    const filteredTodos = todos.filter(todo => {
        if (filter === 'active') return !todo.completed;
        if (filter === 'completed') return todo.completed;
        return true;
    });

    return (
        <main className="min-h-screen bg-gradient-to-br from-blue-50 to-indigo-100 py-12 px-4">
            <div className="max-w-2xl mx-auto">
                <h1 className="text-5xl font-bold text-center mb-12 text-gray-800">
                    My Todo App
                </h1>

                <div className="mb-6">
                    <AddTodo onAdd={addTodo} />
                </div>

                {/* 筛选器 */}
                <div className="flex gap-2 mb-6">
                    {(['all', 'active', 'completed'] as FilterType[]).map(f => (
                        <button
                            key={f}
                            onClick={() => setFilter(f)}
                            className={`px-4 py-2 rounded-lg font-medium transition-colors ${
                                filter === f
                                    ? 'bg-blue-500 text-white'
                                    : 'bg-white text-gray-700 hover:bg-gray-100'
                            }`}
                        >
                            {f.charAt(0).toUpperCase() + f.slice(1)}
                        </button>
                    ))}
                </div>

                {/* Todo列表 */}
                <div className="space-y-2">
                    {filteredTodos.length === 0 ? (
                        <p className="text-center text-gray-500 py-12">
                            No todos yet. Add one above!
                        </p>
                    ) : (
                        filteredTodos.map(todo => (
                            <TodoItem
                                key={todo.id}
                                todo={todo}
                                onToggle={toggleTodo}
                                onDelete={deleteTodo}
                            />
                        ))
                    )}
                </div>

                {/* 统计 */}
                <div className="mt-6 text-center text-gray-600">
                    {todos.filter(t => !t.completed).length} items left
                </div>
            </div>
        </main>
    );
}
```

**✅ 完成标准**:
- [ ] 能添加、删除、标记Todo
- [ ] 数据刷新不丢失
- [ ] 筛选功能正常
- [ ] UI美观且响应式

---

### 📅 Day 7 (Sunday): 部署 + 周总结

#### 上午: 部署到 Vercel

**步骤**:
```bash
# 1. 提交代码到GitHub
git init
git add .
git commit -m "Initial commit: Todo App"
git branch -M main
git remote add origin <your-github-repo>
git push -u origin main

# 2. Vercel部署
# 访问 https://vercel.com
# - 用GitHub登录
# - Import项目
# - 一键部署
# - 获得 https://your-app.vercel.app
```

#### 下午: 周总结和改进

**回顾清单**:
- [ ] TypeScript 基础是否掌握？
- [ ] React 核心概念是否理解？
- [ ] Tailwind CSS 能否快速使用？
- [ ] Todo App 是否完成并部署？

**改进建议**: 让AI帮你添加这些功能
```
1. 编辑Todo
2. 拖拽排序
3. 优先级标记
4. 到期日期
5. 暗黑模式
```

**📚 本周学习资源总结**:
- React官方文档: https://react.dev/learn
- TypeScript手册: https://www.typescriptlang.org/docs/
- Tailwind CSS: https://tailwindcss.com/docs
- Next.js文档: https://nextjs.org/docs

---

## 📅 Week 2: 进阶技能 + 实战项目

### 🎯 本周目标
- ✅ 掌握 Next.js App Router
- ✅ 学会API开发和数据库操作
- ✅ 完成一个全栈CRUD应用
- ✅ 引入组件库（shadcn/ui）

---

### 📅 Day 8 (Monday): Next.js App Router深入

#### 全天 (3小时): 路由和布局

**核心概念**:
```
app/
├── page.tsx              # / 路由
├── layout.tsx            # 根布局
├── about/
│   └── page.tsx          # /about 路由
├── blog/
│   ├── page.tsx          # /blog 路由
│   └── [id]/
│       └── page.tsx      # /blog/:id 动态路由
└── api/
    └── todos/
        └── route.ts      # /api/todos API路由
```

**路由示例**:
```tsx
// app/blog/page.tsx
export default function BlogPage() {
    return <h1>Blog</h1>;
}

// app/blog/[id]/page.tsx
interface PageProps {
    params: { id: string };
}

export default function BlogPost({ params }: PageProps) {
    return <h1>Post {params.id}</h1>;
}

// 使用Link导航
import Link from 'next/link';

<Link href="/blog/123" className="text-blue-500 hover:underline">
    Read Post
</Link>
```

**布局系统**:
```tsx
// app/layout.tsx (根布局，应用于所有页面)
export default function RootLayout({
    children,
}: {
    children: React.ReactNode;
}) {
    return (
        <html lang="en">
            <body>
                <nav className="bg-gray-800 text-white p-4">
                    {/* 全局导航 */}
                </nav>
                {children}
                <footer className="bg-gray-800 text-white p-4">
                    {/* 全局页脚 */}
                </footer>
            </body>
        </html>
    );
}

// app/blog/layout.tsx (嵌套布局，只应用于/blog下)
export default function BlogLayout({
    children,
}: {
    children: React.ReactNode;
}) {
    return (
        <div className="max-w-4xl mx-auto">
            <aside className="w-64 bg-gray-100 p-4">
                {/* 侧边栏 */}
            </aside>
            <main>{children}</main>
        </div>
    );
}
```

**实战练习**: 创建多页面应用
```
项目: 个人博客
页面:
- / (首页)
- /about (关于)
- /blog (博客列表)
- /blog/[id] (博客详情)
- /contact (联系)

要求: 统一导航栏，每个页面不同内容
```

**✅ 完成标准**:
- [ ] 理解App Router文件系统路由
- [ ] 会使用动态路由
- [ ] 理解布局嵌套

---

### 📅 Day 9 (Tuesday): API Routes + 数据获取

#### 上午: API Routes

**创建API**:
```typescript
// app/api/todos/route.ts
import { NextRequest, NextResponse } from 'next/server';

// GET /api/todos
export async function GET(request: NextRequest) {
    // 模拟数据库查询
    const todos = [
        { id: 1, title: "Learn Next.js", completed: false },
        { id: 2, title: "Build App", completed: true }
    ];

    return NextResponse.json({ success: true, data: todos });
}

// POST /api/todos
export async function POST(request: NextRequest) {
    const body = await request.json();

    // 验证
    if (!body.title) {
        return NextResponse.json(
            { success: false, error: "Title is required" },
            { status: 400 }
        );
    }

    // 创建Todo
    const newTodo = {
        id: Date.now(),
        title: body.title,
        completed: false
    };

    return NextResponse.json({ success: true, data: newTodo }, { status: 201 });
}

// app/api/todos/[id]/route.ts
export async function DELETE(
    request: NextRequest,
    { params }: { params: { id: string } }
) {
    const id = params.id;

    // 删除逻辑

    return NextResponse.json({ success: true });
}
```

#### 下午: 数据获取

**使用fetch**:
```typescript
'use client';

import { useEffect, useState } from 'react';

function TodosPage() {
    const [todos, setTodos] = useState([]);
    const [loading, setLoading] = useState(true);

    useEffect(() => {
        fetch('/api/todos')
            .then(res => res.json())
            .then(data => {
                setTodos(data.data);
                setLoading(false);
            })
            .catch(error => console.error(error));
    }, []);

    if (loading) return <div>Loading...</div>;

    return (
        <ul>
            {todos.map(todo => (
                <li key={todo.id}>{todo.title}</li>
            ))}
        </ul>
    );
}
```

**使用 TanStack Query (推荐)**:
```bash
npm install @tanstack/react-query
```

```typescript
// app/providers.tsx
'use client';

import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { useState } from 'react';

export function Providers({ children }: { children: React.ReactNode }) {
    const [queryClient] = useState(() => new QueryClient());

    return (
        <QueryClientProvider client={queryClient}>
            {children}
        </QueryClientProvider>
    );
}

// app/layout.tsx
import { Providers } from './providers';

export default function RootLayout({ children }) {
    return (
        <html>
            <body>
                <Providers>{children}</Providers>
            </body>
        </html>
    );
}

// app/todos/page.tsx
'use client';

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';

function TodosPage() {
    const queryClient = useQueryClient();

    // 查询
    const { data, isLoading, error } = useQuery({
        queryKey: ['todos'],
        queryFn: async () => {
            const res = await fetch('/api/todos');
            return res.json();
        }
    });

    // 变更
    const mutation = useMutation({
        mutationFn: async (newTodo: { title: string }) => {
            const res = await fetch('/api/todos', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(newTodo)
            });
            return res.json();
        },
        onSuccess: () => {
            // 刷新数据
            queryClient.invalidateQueries({ queryKey: ['todos'] });
        }
    });

    if (isLoading) return <div>Loading...</div>;
    if (error) return <div>Error: {error.message}</div>;

    return (
        <div>
            <button onClick={() => mutation.mutate({ title: "New Todo" })}>
                Add Todo
            </button>
            <ul>
                {data.data.map(todo => (
                    <li key={todo.id}>{todo.title}</li>
                ))}
            </ul>
        </div>
    );
}
```

**✅ 完成标准**:
- [ ] 能创建RESTful API
- [ ] 会处理GET/POST/DELETE请求
- [ ] 理解React Query的优势

---

### 📅 Day 10-11 (Wed-Thu): 数据库集成

#### 安装 Prisma
```bash
npm install prisma @prisma/client
npx prisma init
```

#### 配置数据库
```env
# .env
DATABASE_URL="file:./dev.db"  # SQLite (本地开发)
# DATABASE_URL="postgresql://..." # PostgreSQL (生产环境)
```

#### 定义Schema
```prisma
// prisma/schema.prisma
generator client {
  provider = "prisma-client-js"
}

datasource db {
  provider = "sqlite"
  url      = env("DATABASE_URL")
}

model Todo {
  id        Int      @id @default(autoincrement())
  title     String
  completed Boolean  @default(false)
  createdAt DateTime @default(now())
  updatedAt DateTime @updatedAt
}
```

#### 创建数据库
```bash
npx prisma migrate dev --name init
npx prisma generate
```

#### 使用 Prisma Client
```typescript
// lib/prisma.ts
import { PrismaClient } from '@prisma/client';

const globalForPrisma = global as unknown as { prisma: PrismaClient };

export const prisma =
    globalForPrisma.prisma ||
    new PrismaClient({
        log: ['query'],
    });

if (process.env.NODE_ENV !== 'production') globalForPrisma.prisma = prisma;

// app/api/todos/route.ts
import { prisma } from '@/lib/prisma';

export async function GET() {
    const todos = await prisma.todo.findMany({
        orderBy: { createdAt: 'desc' }
    });

    return NextResponse.json({ success: true, data: todos });
}

export async function POST(request: NextRequest) {
    const body = await request.json();

    const todo = await prisma.todo.create({
        data: {
            title: body.title,
            completed: false
        }
    });

    return NextResponse.json({ success: true, data: todo }, { status: 201 });
}

// app/api/todos/[id]/route.ts
export async function DELETE(
    request: NextRequest,
    { params }: { params: { id: string } }
) {
    await prisma.todo.delete({
        where: { id: parseInt(params.id) }
    });

    return NextResponse.json({ success: true });
}

export async function PATCH(
    request: NextRequest,
    { params }: { params: { id: string } }
) {
    const body = await request.json();

    const todo = await prisma.todo.update({
        where: { id: parseInt(params.id) },
        data: body
    });

    return NextResponse.json({ success: true, data: todo });
}
```

**✅ 完成标准**:
- [ ] Prisma安装配置成功
- [ ] 能进行CRUD操作
- [ ] 数据持久化到数据库

---

### 📅 Day 12-13 (Fri-Sat): shadcn/ui组件库

#### 安装 shadcn/ui
```bash
npx shadcn-ui@latest init

# 选择配置
✔ Would you like to use TypeScript? … yes
✔ Which style would you like to use? › Default
✔ Which color would you like to use as base color? › Slate
✔ Where is your global CSS file? … app/globals.css
✔ Would you like to use CSS variables for colors? … yes
✔ Where is your tailwind.config.js located? … tailwind.config.ts
✔ Configure the import alias for components: … @/components
✔ Configure the import alias for utils: … @/lib/utils
```

#### 添加组件
```bash
npx shadcn-ui@latest add button
npx shadcn-ui@latest add card
npx shadcn-ui@latest add input
npx shadcn-ui@latest add dialog
npx shadcn-ui@latest add form
npx shadcn-ui@latest add table
```

#### 使用组件
```tsx
import { Button } from "@/components/ui/button";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";

function MyComponent() {
    return (
        <Card>
            <CardHeader>
                <CardTitle>Hello</CardTitle>
            </CardHeader>
            <CardContent>
                <div className="flex gap-2">
                    <Input placeholder="Enter text..." />
                    <Button>Submit</Button>
                </div>
            </CardContent>
        </Card>
    );
}
```

**实战项目**: 用shadcn/ui重构Todo App

**✅ 完成标准**:
- [ ] shadcn/ui配置成功
- [ ] 能使用各种组件
- [ ] UI更加专业美观

---

### 📅 Day 14 (Sunday): 完整CRUD项目

#### 项目: 笔记管理应用

**功能**:
- ✅ 创建、读取、更新、删除笔记
- ✅ Markdown编辑器
- ✅ 标签系统
- ✅ 搜索功能
- ✅ 响应式设计

**技术栈**:
- Next.js 14
- Prisma + SQLite
- shadcn/ui
- TanStack Query
- react-markdown

**提示词给AI**:
```
创建一个笔记管理应用，技术栈：Next.js 14 + Prisma + shadcn/ui + TanStack Query

数据模型:
- Note: id, title, content(Markdown), tags[], createdAt, updatedAt

功能:
1. 笔记CRUD (创建、读取、编辑、删除)
2. Markdown实时预览
3. 标签筛选
4. 搜索功能
5. 响应式布局

请提供完整代码和详细说明
```

**✅ 完成标准**:
- [ ] 完整CRUD功能
- [ ] 数据库持久化
- [ ] Markdown支持
- [ ] 部署上线

---

## 📅 Week 3: 高级特性 + 独立项目

### 🎯 本周目标
- ✅ 学习认证和授权
- ✅ 掌握状态管理
- ✅ 学习实时通信
- ✅ 完成一个完整的独立项目

---

### 📅 Day 15-16 (Mon-Tue): 认证授权

#### NextAuth.js 集成

**安装**:
```bash
npm install next-auth @auth/prisma-adapter
```

**配置**:
```typescript
// app/api/auth/[...nextauth]/route.ts
import NextAuth from "next-auth";
import GithubProvider from "next-auth/providers/github";
import { PrismaAdapter } from "@auth/prisma-adapter";
import { prisma } from "@/lib/prisma";

const handler = NextAuth({
    adapter: PrismaAdapter(prisma),
    providers: [
        GithubProvider({
            clientId: process.env.GITHUB_ID!,
            clientSecret: process.env.GITHUB_SECRET!,
        }),
    ],
    callbacks: {
        session: async ({ session, user }) => {
            session.user.id = user.id;
            return session;
        },
    },
});

export { handler as GET, handler as POST };
```

**使用认证**:
```tsx
'use client';

import { useSession, signIn, signOut } from "next-auth/react";

export default function Profile() {
    const { data: session, status } = useSession();

    if (status === "loading") return <div>Loading...</div>;

    if (!session) {
        return (
            <button onClick={() => signIn("github")}>
                Sign in with GitHub
            </button>
        );
    }

    return (
        <div>
            <p>Welcome, {session.user?.name}!</p>
            <button onClick={() => signOut()}>Sign out</button>
        </div>
    );
}
```

**保护路由**:
```tsx
// middleware.ts
export { default } from "next-auth/middleware";

export const config = { matcher: ["/dashboard/:path*"] };
```

**✅ 完成标准**:
- [ ] NextAuth.js配置成功
- [ ] 能登录登出
- [ ] 受保护路由生效

---

### 📅 Day 17 (Wednesday): 状态管理

#### Zustand (推荐，最简单)

**安装**:
```bash
npm install zustand
```

**创建Store**:
```typescript
// store/useUserStore.ts
import { create } from 'zustand';

interface UserState {
    user: User | null;
    setUser: (user: User) => void;
    logout: () => void;
}

export const useUserStore = create<UserState>((set) => ({
    user: null,
    setUser: (user) => set({ user }),
    logout: () => set({ user: null }),
}));

// 使用
function Profile() {
    const { user, setUser, logout } = useUserStore();

    return <div>{user?.name}</div>;
}
```

**持久化**:
```typescript
import { create } from 'zustand';
import { persist } from 'zustand/middleware';

export const useStore = create(
    persist(
        (set) => ({
            count: 0,
            increment: () => set((state) => ({ count: state.count + 1 })),
        }),
        {
            name: 'app-storage', // localStorage key
        }
    )
);
```

**✅ 完成标准**:
- [ ] 理解Zustand的简单性
- [ ] 能创建全局状态
- [ ] 会用持久化中间件

---

### 📅 Day 18 (Thursday): 实时通信

#### Server-Sent Events (SSE)

**API端**:
```typescript
// app/api/events/route.ts
export async function GET() {
    const encoder = new TextEncoder();

    const stream = new ReadableStream({
        async start(controller) {
            const sendEvent = (data: any) => {
                controller.enqueue(
                    encoder.encode(`data: ${JSON.stringify(data)}\n\n`)
                );
            };

            // 每秒发送一次
            const interval = setInterval(() => {
                sendEvent({ time: new Date().toISOString() });
            }, 1000);

            // 清理
            return () => clearInterval(interval);
        },
    });

    return new Response(stream, {
        headers: {
            'Content-Type': 'text/event-stream',
            'Cache-Control': 'no-cache',
            'Connection': 'keep-alive',
        },
    });
}
```

**客户端**:
```tsx
'use client';

import { useEffect, useState } from 'react';

export default function RealtimePage() {
    const [messages, setMessages] = useState<string[]>([]);

    useEffect(() => {
        const eventSource = new EventSource('/api/events');

        eventSource.onmessage = (event) => {
            const data = JSON.parse(event.data);
            setMessages(prev => [...prev, data.time]);
        };

        return () => eventSource.close();
    }, []);

    return (
        <ul>
            {messages.map((msg, i) => (
                <li key={i}>{msg}</li>
            ))}
        </ul>
    );
}
```

**✅ 完成标准**:
- [ ] 理解SSE工作原理
- [ ] 能实现实时推送

---

### 📅 Day 19-21 (Fri-Sun): 独立项目

#### 项目选择（三选一）

**选项1: 全栈任务管理系统**
```
功能:
- 用户认证 (NextAuth.js)
- 项目和任务CRUD
- 拖拽排序 (dnd-kit)
- 实时协作 (SSE)
- 评论系统
- 文件上传
- 搜索和筛选
- Dashboard统计

技术栈:
- Next.js 14 + TypeScript
- Prisma + PostgreSQL
- NextAuth.js
- Zustand
- shadcn/ui + dnd-kit
- TanStack Query
```

**选项2: eBPF可视化工具**
```
功能:
- eBPF程序代码编辑器 (Monaco Editor)
- BPF Map可视化
- 数据流图展示 (React Flow)
- 日志实时查看
- 性能指标Dashboard
- 代码模板库

技术栈:
- Next.js 14
- @monaco-editor/react
- react-flow-renderer
- Recharts
- shadcn/ui

优势: 结合你的专业领域！
```

**选项3: AI聊天应用**
```
功能:
- 用户认证
- 多会话管理
- Markdown渲染
- 代码高亮
- 流式响应
- 历史记录
- 分享功能

技术栈:
- Next.js 14
- OpenAI API
- Prisma
- NextAuth.js
- react-markdown
- Vercel AI SDK
```

#### 开发流程

**Day 19**: 项目设计
- 需求分析
- 数据模型设计
- UI设计（Figma或手绘）
- 技术选型

**Day 20**: 核心功能开发
- 数据库Schema
- API开发
- 核心UI组件
- 状态管理

**Day 21**: 完善和部署
- 细节优化
- 错误处理
- 响应式适配
- 部署到Vercel
- 文档编写

**AI辅助技巧**:
```
# 分步骤让AI帮你
1. "帮我设计任务管理系统的Prisma Schema"
2. "创建任务CRUD的API Routes"
3. "设计任务列表的UI组件"
4. "实现拖拽排序功能"
5. "添加实时协作功能"

# 遇到问题
"报错: [错误信息]，如何解决？"
"这个功能性能很差，如何优化？"
"如何实现[具体功能]？"
```

**✅ 完成标准**:
- [ ] 项目完整可用
- [ ] 代码结构清晰
- [ ] 部署成功
- [ ] 写了README文档

---

## 📚 学习资源总结

### 官方文档（最权威）
- **React**: https://react.dev/learn
- **Next.js**: https://nextjs.org/docs
- **TypeScript**: https://www.typescriptlang.org/docs/
- **Tailwind CSS**: https://tailwindcss.com/docs
- **Prisma**: https://www.prisma.io/docs
- **shadcn/ui**: https://ui.shadcn.com/

### 视频教程
- **YouTube**: "Next.js 14 Full Course" by CodeWithAntonio
- **B站**: 搜"Next.js 14完整教程"

### 实战项目参考
- **GitHub**: 搜 "next.js-14 project"
- **Vercel**: https://vercel.com/templates (官方模板)

### AI工具最佳实践
- **v0.dev**: Vercel的AI生成UI组件
- **Cursor**: AI配对编程
- **Claude Code**: 代码生成和解释

---

## 🎯 3周后检验清单

### 技能清单
- [ ] TypeScript基础语法熟练
- [ ] React核心概念理解
- [ ] Next.js App Router使用
- [ ] Tailwind CSS快速布局
- [ ] API开发能力
- [ ] 数据库操作（Prisma）
- [ ] 组件库使用（shadcn/ui）
- [ ] 状态管理（Zustand）
- [ ] 认证授权（NextAuth.js）
- [ ] 部署发布（Vercel）

### 项目清单
- [ ] Todo App (Week 1)
- [ ] 笔记管理 (Week 2)
- [ ] 独立完整项目 (Week 3)

### 能力清单
- [ ] 能独立搭建Next.js项目
- [ ] 能完成CRUD功能
- [ ] 能对接第三方API
- [ ] 能设计响应式UI
- [ ] 能使用AI快速解决问题
- [ ] 能阅读和理解React代码
- [ ] 能部署项目到生产环境

---

## 💡 学习建议

### 1. 不要追求完美
- 能跑就行，慢慢优化
- 先完成，再完美
- 不懂的地方问AI

### 2. 充分利用AI
- 让AI写样板代码
- 你负责业务逻辑
- 遇到bug直接问AI
- 看不懂的代码让AI解释

### 3. 从后端视角理解前端
- 组件 = 函数
- Props = 函数参数
- State = 变量
- useEffect = cleanup/defer
- API Routes = 你熟悉的后端

### 4. 实战大于理论
- 做3个项目 > 看10个教程
- 每个项目解决一个实际问题
- 项目要能展示给别人看

### 5. 保持学习节奏
- 每天2-3小时
- 周末4-6小时
- 不要一次性学太多
- 及时复习和总结

---

## 🚀 立即开始

```bash
# 创建第一个项目
npx create-next-app@latest my-first-app --typescript --tailwind --app

cd my-first-app
npm run dev

# 打开 http://localhost:3000
# 开始你的前端之旅！
```

**第一个任务**: 让Claude/Cursor帮你把首页改成计数器，理解每一行代码。

**记住**: 有C++/Go基础 + AI工具 = 3周掌握前端开发！💪

加油！🎉
