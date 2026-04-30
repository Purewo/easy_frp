import React, { useState, useEffect } from 'react';
import { X, Search, Check, Trash2, Download, Upload, AlertCircle, Loader2, Globe, FileText } from 'lucide-react';
import { movieService } from '../api';
import { ResourceSubtitleItem } from '../types';
import { toast } from '../utils';

interface SubtitleManagerDialogProps {
  resourceId: string;
  initialSubtitles: ResourceSubtitleItem[];
  defaultSubtitleId?: string;
  movieTitle: string;
  season?: number;
  episode?: number;
  onClose: () => void;
  onSubtitlesChange: (items: ResourceSubtitleItem[], defaultId?: string) => void;
}

export const SubtitleManagerDialog: React.FC<SubtitleManagerDialogProps> = ({ 
  resourceId, 
  initialSubtitles, 
  defaultSubtitleId,
  movieTitle, 
  season, 
  episode,
  onClose,
  onSubtitlesChange
}) => {
  const [subtitles, setSubtitles] = useState<ResourceSubtitleItem[]>(initialSubtitles || []);
  const [activeDefaultId, setActiveDefaultId] = useState<string | undefined>(defaultSubtitleId);
  const [isSearching, setIsSearching] = useState(false);
  const [searchResults, setSearchResults] = useState<any[]>([]);
  const [searchKeyword, setSearchKeyword] = useState('');
  const [isProcessing, setIsProcessing] = useState<string | null>(null);

  useEffect(() => {
    // Generate a good default keyword
    let ks = movieTitle;
    if (season && episode) {
        // Just the title since the backend sorts by season/episode automatically according to OpenAPI comments
        ks = movieTitle;
    }
    setSearchKeyword(ks);
  }, [movieTitle, season, episode]);

  const handleSearch = async () => {
    if (!searchKeyword.trim()) return;
    setIsSearching(true);
    setSearchResults([]);
    try {
      const res = await movieService.searchOnlineSubtitles(resourceId, searchKeyword);
      setSearchResults(res?.items || []);
    } catch (e: any) {
      toast.error('Failed to search subtitles: ' + (e.message || String(e)));
    } finally {
      setIsSearching(false);
    }
  };

  const handleBind = async (candidateId: string) => {
    setIsProcessing(candidateId);
    try {
      const res = await movieService.bindOnlineSubtitle(resourceId, candidateId);
      // Backend returns updated playback.subtitles
      if (res && res.items) {
          setSubtitles(res.items);
          if (res.default_subtitle_id) setActiveDefaultId(res.default_subtitle_id);
          onSubtitlesChange(res.items, res.default_subtitle_id);
      } else {
          toast.success('Subtitle bound, please refresh page.');
          onClose();
      }
    } catch (e: any) {
      toast.error('Failed to bind subtitle: ' + (e.message || String(e)));
    } finally {
      setIsProcessing(null);
    }
  };

  const handleDelete = async (subtitleId: string) => {
    try {
      // The openapi delete does not return updated object but just 200
      await movieService.deleteSubtitle(resourceId, subtitleId);
      const updated = subtitles.filter(s => s.id !== subtitleId);
      setSubtitles(updated);
      onSubtitlesChange(updated, activeDefaultId === subtitleId ? undefined : activeDefaultId);
    } catch (e: any) {
      toast.error('Failed to delete subtitle: ' + (e.message || String(e)));
    }
  };

  const handleSetDefault = async (subtitleId: string) => {
    try {
      await movieService.setDefaultSubtitle(resourceId, subtitleId);
      setActiveDefaultId(subtitleId);
      onSubtitlesChange(subtitles, subtitleId);
    } catch (e: any) {
      toast.error('Failed to set default: ' + (e.message || String(e)));
    }
  };

  const currentBoundSubtitles = subtitles.filter(s => s.source === 'online_bound' || s.source === 'manual_upload');
  const localSubtitles = subtitles.filter(s => s.source === 'sidecar');

  return (
    <div className="fixed inset-0 bg-black/80 backdrop-blur-sm z-[200] flex items-center justify-center p-4 animate-in fade-in">
      <div className="bg-gray-900 border border-white/10 w-full max-w-2xl rounded-sm shadow-2xl flex flex-col h-[80vh] overflow-hidden">
        {/* Header */}
        <div className="flex items-center justify-between p-4 border-b border-white/10 shrink-0">
          <div className="flex items-center gap-2">
            <Globe className="text-primary w-5 h-5" />
            <h2 className="text-lg font-['Orbitron'] font-bold tracking-wider text-white">SUBTITLE MANAGER</h2>
          </div>
          <button onClick={onClose} className="p-2 text-gray-400 hover:text-white transition-colors hover:bg-white/10 rounded-sm">
            <X size={20} />
          </button>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-y-auto p-6 space-y-8 custom-scrollbar">
          
          {/* Current Subtitles */}
          <div className="space-y-4">
            <h3 className="text-sm font-bold text-gray-400 uppercase tracking-widest border-b border-gray-800 pb-2">Active Subtitles</h3>
            
            {subtitles.length === 0 ? (
                <div className="text-center py-6 bg-black/20 rounded-sm border border-white/5 text-gray-500 text-sm">
                    No subtitles bound or found locally.
                </div>
            ) : (
                <div className="space-y-2">
                    {subtitles.map(sub => {
                        const isOnline = sub.source === 'online_bound' || sub.source === 'manual_upload';
                        const isDefault = activeDefaultId === sub.id || sub.is_default;
                        
                        return (
                            <div key={sub.id} className="flex items-center justify-between bg-black/40 border border-white/10 p-3 rounded-sm group hover:border-primary/30 transition-colors">
                                <div className="flex items-center gap-3 overflow-hidden">
                                    <FileText size={16} className={isOnline ? 'text-blue-400' : 'text-gray-400'} />
                                    <div className="flex flex-col truncate">
                                        <span className="text-sm text-gray-200 truncate">{(sub.label && sub.label.startsWith('Unknown')) ? (sub.filename || sub.label || sub.title) : (sub.label || sub.filename || sub.title)}</span>
                                        <span className="text-[10px] text-gray-500 uppercase tracking-wide flex items-center gap-2">
                                            <span>{sub.language || 'UNKNOWN LANG'}</span>
                                            <span className="w-1 h-1 bg-gray-700 mx-1 rounded-full"></span>
                                            <span>{sub.format}</span>
                                            <span className="w-1 h-1 bg-gray-700 mx-1 rounded-full"></span>
                                            <span>{sub.source}</span>
                                        </span>
                                    </div>
                                </div>
                                <div className="flex items-center gap-2 shrink-0">
                                    {isDefault ? (
                                        <span className="text-xs bg-primary/20 text-primary px-2 py-1 rounded-sm border border-primary/20 flex items-center gap-1 font-bold">
                                            <Check size={12} /> DEFAULT
                                        </span>
                                    ) : (
                                        <button 
                                            onClick={() => handleSetDefault(sub.id)}
                                            className="text-xs text-gray-400 hover:text-white px-2 py-1 rounded-sm hover:bg-white/10 transition-colors opacity-0 group-hover:opacity-100"
                                        >
                                            Set Default
                                        </button>
                                    )}
                                    {isOnline && (
                                        <button 
                                            onClick={() => handleDelete(sub.id)}
                                            className="p-1.5 text-gray-500 hover:text-red-400 hover:bg-red-400/10 rounded-sm transition-colors"
                                            title="Remove"
                                        >
                                            <Trash2 size={16} />
                                        </button>
                                    )}
                                </div>
                            </div>
                        );
                    })}
                </div>
            )}
          </div>

          {/* Online Search */}
          <div className="space-y-4">
            <h3 className="text-sm font-bold text-gray-400 uppercase tracking-widest border-b border-gray-800 pb-2">Online Search</h3>
            
            <div className="flex gap-2">
                <input 
                    type="text" 
                    value={searchKeyword}
                    onChange={e => setSearchKeyword(e.target.value)}
                    onKeyDown={e => e.key === 'Enter' && handleSearch()}
                    placeholder="Search keywords..."
                    className="flex-1 bg-black/40 border border-white/20 text-white px-3 py-2 rounded-sm focus:outline-none focus:border-primary transition-colors text-sm"
                />
                <button 
                    onClick={handleSearch}
                    disabled={isSearching}
                    className="bg-primary/20 text-primary border border-primary/50 hover:bg-primary hover:text-black font-bold px-4 py-2 rounded-sm flex items-center gap-2 transition-all disabled:opacity-50 disabled:cursor-not-allowed"
                >
                    {isSearching ? <Loader2 size={16} className="animate-spin" /> : <Search size={16} />}
                    SEARCH
                </button>
            </div>

            {/* Results */}
            {searchResults.length > 0 && (
                <div className="space-y-2 mt-4">
                    {searchResults.map((res: any) => (
                        <div key={res.id} className="flex flex-col sm:flex-row sm:items-center justify-between bg-black/20 border border-white/5 p-3 rounded-sm hover:bg-white/5 transition-colors gap-3">
                            <div className="flex flex-col overflow-hidden max-w-[80%]">
                                <span className="text-sm text-gray-200 truncate" title={res.title || res.name}>{res.title || res.name}</span>
                                <div className="flex items-center gap-2 mt-1 text-[10px] text-gray-500">
                                    <span className="bg-white/10 px-1.5 py-0.5 rounded-sm uppercase">{res.source || res.id?.split(':')[0]}</span>
                                    {res.rating && <span className="text-yellow-500 w-10 truncate">Rating: {res.rating}</span>}
                                    {res.language && <span>{res.language}</span>}
                                    {res.format && <span>{res.format}</span>}
                                </div>
                            </div>
                            <button 
                                onClick={() => handleBind(res.id)}
                                disabled={isProcessing === res.id}
                                className="self-end sm:self-auto shrink-0 flex items-center gap-2 bg-white/10 hover:bg-white/20 text-white px-3 py-1.5 rounded-sm text-xs font-bold transition-colors disabled:opacity-50"
                            >
                                {isProcessing === res.id ? <Loader2 size={14} className="animate-spin" /> : <Download size={14} />}
                                BIND
                            </button>
                        </div>
                    ))}
                </div>
            )}
            
            {!isSearching && searchResults.length === 0 && searchKeyword && (
                <div className="text-xs text-gray-500 flex items-center gap-2 bg-black/20 p-3 rounded-sm">
                    <AlertCircle size={14} />
                    Try searching to find subtitles via SubHD / SrtKu.
                </div>
            )}

            {/* Upload Manually Option */}
            <div className="mt-8 pt-4 border-t border-white/5 flex flex-col items-center justify-center p-6 bg-black/20 border border-dashed border-white/20 rounded-sm">
                <Upload size={24} className="text-gray-500 mb-2" />
                <p className="text-sm text-gray-400 mb-4">Have your own subtitle file?</p>
                <label className="cursor-pointer bg-white/10 hover:bg-white/20 text-white px-4 py-2 rounded-sm text-xs font-bold transition-colors">
                    <span>SELECT FILE</span>
                    <input 
                        type="file" 
                        accept=".srt,.ass,.ssa,.vtt,.sub,.sup,.zip,.7z,.rar" 
                        className="hidden" 
                        onChange={async (e) => {
                            const file = e.target.files?.[0];
                            if (!file) return;
                            try {
                                setIsProcessing('upload');
                                const res = await movieService.uploadSubtitle(resourceId, file, true);
                                if (res && res.items) {
                                    setSubtitles(res.items);
                                    if (res.default_subtitle_id) setActiveDefaultId(res.default_subtitle_id);
                                    onSubtitlesChange(res.items, res.default_subtitle_id);
                                }
                                toast.success('Uploaded successfully');
                            } catch (error: any) {
                                toast.error('Upload failed: ' + error.message);
                            } finally {
                                setIsProcessing(null);
                            }
                        }}
                    />
                </label>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};
