import { useState, useEffect } from 'react';
import { useProjectStore } from '@/stores/projectStore';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog';
import { ScrollArea } from '@/components/ui/scroll-area';
import { cn } from '@/lib/utils';
import { Plus, Trash2, Save, FileText } from 'lucide-react';
import type { Entity, Field, FieldType } from '@/types/dtrules';

const FIELD_TYPES: FieldType[] = ['string', 'integer', 'boolean', 'date', 'double', 'entity', 'array'];
const ACCESS_TYPES = ['r', 'rw', 'w'];

export function EDDEditor() {
  const { entities, currentEntity, selectEntity, updateEntity, createEntity, deleteEntity } = useProjectStore();
  const [editedEntity, setEditedEntity] = useState<Entity | null>(null);
  const [newEntityDialogOpen, setNewEntityDialogOpen] = useState(false);
  const [newEntityName, setNewEntityName] = useState('');

  useEffect(() => {
    if (currentEntity) {
      setEditedEntity({ ...currentEntity, fields: [...currentEntity.fields] });
    } else {
      setEditedEntity(null);
    }
  }, [currentEntity]);

  const handleFieldChange = (index: number, key: keyof Field, value: string) => {
    if (!editedEntity) return;
    const newFields = [...editedEntity.fields];
    newFields[index] = { ...newFields[index], [key]: value };
    setEditedEntity({ ...editedEntity, fields: newFields });
  };

  const handleAddField = () => {
    if (!editedEntity) return;
    const newField: Field = {
      name: 'newField',
      type: 'string',
      subtype: '',
      access: 'rw',
      input: '',
      defaultValue: '',
      comment: '',
    };
    setEditedEntity({ ...editedEntity, fields: [...editedEntity.fields, newField] });
  };

  const handleDeleteField = (index: number) => {
    if (!editedEntity) return;
    const newFields = editedEntity.fields.filter((_, i) => i !== index);
    setEditedEntity({ ...editedEntity, fields: newFields });
  };

  const handleSave = async () => {
    if (!editedEntity) return;
    await updateEntity(editedEntity);
  };

  const handleCreateEntity = async () => {
    if (!newEntityName) return;
    const success = await createEntity({
      name: newEntityName,
      access: 'rw',
      comment: '',
      fields: [],
    });
    if (success) {
      setNewEntityDialogOpen(false);
      setNewEntityName('');
      selectEntity(newEntityName);
    }
  };

  const handleDeleteEntity = async () => {
    if (!currentEntity) return;
    if (confirm(`Delete entity "${currentEntity.name}"?`)) {
      await deleteEntity(currentEntity.name);
    }
  };

  if (!entities.length) {
    return (
      <div className="h-full flex items-center justify-center text-muted-foreground" data-tutorial="entity-editor">
        <p>No entities loaded. Open a project to see entities.</p>
      </div>
    );
  }

  return (
    <div className="h-full flex" data-tutorial="entity-editor">
      {/* Entity list sidebar */}
      <div className="w-48 border-r border-border/50 flex flex-col" data-tutorial="entity-list">
        <div className="p-3 border-b border-border/50 bg-gradient-to-r from-blue-500/10 to-transparent flex items-center justify-between">
          <span className="text-sm font-semibold text-foreground/80">Entities</span>
          <Dialog open={newEntityDialogOpen} onOpenChange={setNewEntityDialogOpen}>
            <DialogTrigger asChild>
              <Button variant="ghost" size="icon" className="h-6 w-6">
                <Plus className="h-4 w-4" />
              </Button>
            </DialogTrigger>
            <DialogContent>
              <DialogHeader>
                <DialogTitle>Create Entity</DialogTitle>
                <DialogDescription>Enter a name for the new entity.</DialogDescription>
              </DialogHeader>
              <div className="grid gap-4 py-4">
                <div className="grid gap-2">
                  <Label htmlFor="entityName">Entity Name</Label>
                  <Input
                    id="entityName"
                    value={newEntityName}
                    onChange={(e) => setNewEntityName(e.target.value)}
                    onKeyDown={(e) => e.key === 'Enter' && handleCreateEntity()}
                  />
                </div>
              </div>
              <DialogFooter>
                <Button onClick={handleCreateEntity} disabled={!newEntityName}>
                  Create
                </Button>
              </DialogFooter>
            </DialogContent>
          </Dialog>
        </div>
        <ScrollArea className="flex-1">
          {entities.map((entity) => (
            <div
              key={entity.name}
              className={cn(
                "px-3 py-2 text-sm cursor-pointer hover:bg-accent",
                currentEntity?.name === entity.name && "bg-accent"
              )}
              onClick={() => selectEntity(entity.name)}
            >
              {entity.name}
            </div>
          ))}
        </ScrollArea>
      </div>

      {/* Entity editor */}
      <div className="flex-1 flex flex-col" data-tutorial="entity-fields">
        {editedEntity ? (
          <>
            {/* Entity header */}
            <div className="p-4 border-b border-border/50 bg-gradient-to-r from-muted/30 via-transparent to-muted/30 flex items-center gap-4">
              <div className="flex-1 flex items-center gap-4">
                <div className="grid gap-1">
                  <Label className="text-xs text-muted-foreground">Entity Name</Label>
                  <Input
                    value={editedEntity.name}
                    onChange={(e) => setEditedEntity({ ...editedEntity, name: e.target.value })}
                    className="h-8"
                  />
                </div>
                <div className="grid gap-1">
                  <Label className="text-xs text-muted-foreground">Access</Label>
                  <Select
                    value={editedEntity.access}
                    onValueChange={(v) => setEditedEntity({ ...editedEntity, access: v })}
                  >
                    <SelectTrigger className="w-20 h-8">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {ACCESS_TYPES.map((type) => (
                        <SelectItem key={type} value={type}>{type}</SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
                <div className="grid gap-1 flex-1">
                  <Label className="text-xs text-muted-foreground">Comment</Label>
                  <Input
                    value={editedEntity.comment}
                    onChange={(e) => setEditedEntity({ ...editedEntity, comment: e.target.value })}
                    className="h-8"
                  />
                </div>
              </div>
              <Button variant="outline" size="sm" onClick={handleDeleteEntity}>
                <Trash2 className="h-4 w-4" />
              </Button>
              <Button size="sm" onClick={handleSave} className="bg-gradient-to-r from-blue-600 to-purple-600 hover:from-blue-500 hover:to-purple-500 text-white border-0">
                <Save className="h-4 w-4 mr-2" />
                Save
              </Button>
            </div>

            {/* Fields table */}
            <div className="flex-1 overflow-auto">
              <table className="w-full text-sm">
                <thead className="bg-gradient-to-r from-muted via-muted/80 to-muted sticky top-0">
                  <tr>
                    <th className="text-left p-2 font-medium">Name</th>
                    <th className="text-left p-2 font-medium w-28">Type</th>
                    <th className="text-left p-2 font-medium w-28">Subtype</th>
                    <th className="text-left p-2 font-medium w-20">Access</th>
                    <th className="text-left p-2 font-medium w-20">Input</th>
                    <th className="text-left p-2 font-medium w-32">Default</th>
                    <th className="text-left p-2 font-medium">Comment</th>
                    <th className="w-10"></th>
                  </tr>
                </thead>
                <tbody>
                  {editedEntity.fields.map((field, index) => (
                    <tr key={index} className="border-b border-border hover:bg-muted/50">
                      <td className="p-1">
                        <Input
                          value={field.name}
                          onChange={(e) => handleFieldChange(index, 'name', e.target.value)}
                          className="h-8"
                        />
                      </td>
                      <td className="p-1">
                        <Select
                          value={field.type}
                          onValueChange={(v) => handleFieldChange(index, 'type', v)}
                        >
                          <SelectTrigger className="h-8">
                            <SelectValue />
                          </SelectTrigger>
                          <SelectContent>
                            {FIELD_TYPES.map((type) => (
                              <SelectItem key={type} value={type}>{type}</SelectItem>
                            ))}
                          </SelectContent>
                        </Select>
                      </td>
                      <td className="p-1">
                        <Input
                          value={field.subtype}
                          onChange={(e) => handleFieldChange(index, 'subtype', e.target.value)}
                          className="h-8"
                          disabled={field.type !== 'entity' && field.type !== 'array'}
                        />
                      </td>
                      <td className="p-1">
                        <Select
                          value={field.access}
                          onValueChange={(v) => handleFieldChange(index, 'access', v)}
                        >
                          <SelectTrigger className="h-8">
                            <SelectValue />
                          </SelectTrigger>
                          <SelectContent>
                            {ACCESS_TYPES.map((type) => (
                              <SelectItem key={type} value={type}>{type}</SelectItem>
                            ))}
                          </SelectContent>
                        </Select>
                      </td>
                      <td className="p-1">
                        <Input
                          value={field.input}
                          onChange={(e) => handleFieldChange(index, 'input', e.target.value)}
                          className="h-8"
                        />
                      </td>
                      <td className="p-1">
                        <Input
                          value={field.defaultValue}
                          onChange={(e) => handleFieldChange(index, 'defaultValue', e.target.value)}
                          className="h-8"
                        />
                      </td>
                      <td className="p-1">
                        <Input
                          value={field.comment}
                          onChange={(e) => handleFieldChange(index, 'comment', e.target.value)}
                          className="h-8"
                        />
                      </td>
                      <td className="p-1">
                        <Button
                          variant="ghost"
                          size="icon"
                          className="h-8 w-8"
                          onClick={() => handleDeleteField(index)}
                        >
                          <Trash2 className="h-4 w-4" />
                        </Button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>

            {/* Add field button */}
            <div className="p-3 border-t border-border/50 bg-gradient-to-r from-muted/20 via-transparent to-muted/20">
              <Button variant="outline" size="sm" onClick={handleAddField} className="border-blue-500/30 hover:border-blue-500/50 hover:bg-blue-500/10">
                <Plus className="h-4 w-4 mr-2" />
                Add Field
              </Button>
            </div>
          </>
        ) : (
          <div className="h-full flex items-center justify-center">
            <div className="text-center">
              <div className="w-16 h-16 mx-auto mb-4 rounded-2xl bg-gradient-to-br from-blue-500/20 to-purple-500/20 flex items-center justify-center">
                <FileText className="h-8 w-8 text-blue-400/60" />
              </div>
              <p className="text-muted-foreground">Select an entity to edit</p>
              <p className="text-xs text-muted-foreground/60 mt-1">Choose from the list on the left</p>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
