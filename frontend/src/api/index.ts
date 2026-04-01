// API 函数封装 - 直接导入 Wails 生成的绑定

import {
  GetAllNotes,
  CreateNote,
  UpdateNote,
  DeleteNote,
  GetNoteCategories,
  GetChatHistory,
  SendChatMessage,
  ClearChatHistory,
  GetSetting,
  SetSetting,
  GetAllSettings,
  CheckLeannStatus,
  RebuildIndex,
  TestLLMConnection,
  GetAppDir,
} from '../../wailsjs/go/main/App'

export {
  // Notes
  GetAllNotes,
  CreateNote,
  UpdateNote,
  DeleteNote,
  GetNoteCategories,
  // Chat
  GetChatHistory,
  SendChatMessage,
  ClearChatHistory,
  // Settings
  GetSetting,
  SetSetting,
  GetAllSettings,
  // LEANN
  CheckLeannStatus,
  RebuildIndex,
  TestLLMConnection,
  GetAppDir,
}
