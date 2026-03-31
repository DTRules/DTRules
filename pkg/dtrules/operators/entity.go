// Copyright 2004-2011 DTRules.com, Inc.
// Copyright 2024 Paul Snow
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package operators

import (
	"github.com/DTRules/DTRules/pkg/dtrules"
)

func init() {
	Register("entitypush", opEntityPush)
	Register("entitypop", opEntityPop)
	Register("def", opDef)
	Register("InContext", opInContext)
	Alias("InContext", "incontext")
	Register("newentity", opNewEntity)
	Register("entityname", opEntityName)
	Register("entityid", opEntityID)
	Register("req", opReq)
	Register("isdefined", opIsDefined)
	Alias("isdefined", "wheredef")
}

// opEntityPush: ( entity -- ) pushes entity onto entity stack
func opEntityPush(state dtrules.State) error {
	obj, err := state.DataPop()
	if err != nil {
		return err
	}
	entity, err := obj.REntityValue()
	if err != nil {
		return err
	}
	return state.EntityPush(entity)
}

// opEntityPop: ( -- entity ) pops entity from entity stack
func opEntityPop(state dtrules.State) error {
	entity, err := state.EntityPop()
	if err != nil {
		return err
	}
	return state.DataPush(entity.(dtrules.Object))
}

// opDef: ( value name -- ) assigns value to name in context
func opDef(state dtrules.State) error {
	nameObj, err := state.DataPop()
	if err != nil {
		return err
	}
	value, err := state.DataPop()
	if err != nil {
		return err
	}
	name, err := nameObj.RNameValue()
	if err != nil {
		return err
	}
	ok, err := state.Def(name, value, true)
	if err != nil {
		return err
	}
	if !ok {
		return dtrules.UndefinedError("def", name.StringValue()+" is undefined")
	}
	return nil
}

// opInContext: ( entityname -- boolean ) checks if entity is in context
func opInContext(state dtrules.State) error {
	nameObj, err := state.DataPop()
	if err != nil {
		return err
	}

	var name *dtrules.RName
	if nameObj.Type() == dtrules.TypeString {
		name = dtrules.GetRName(nameObj.StringValue())
		if name == nil {
			return dtrules.TypeCheckError("InContext", "invalid name syntax: "+nameObj.StringValue())
		}
	} else {
		name, err = nameObj.RNameValue()
		if err != nil {
			return err
		}
	}

	// Save all entities while searching
	var savedEntities []dtrules.Entity
	found := false

	// Pop and save all entities while searching
	for {
		entity, err := state.EntityPop()
		if err != nil {
			break
		}
		savedEntities = append(savedEntities, entity)
		eq, eqErr := entity.GetName().Equals(name)
		if eqErr == nil && eq {
			found = true
			break
		}
	}

	// Restore all saved entities in reverse order (to preserve original order)
	for i := len(savedEntities) - 1; i >= 0; i-- {
		if err := state.EntityPush(savedEntities[i]); err != nil {
			return err
		}
	}

	return state.DataPush(dtrules.GetRBoolean(found))
}

// opNewEntity: ( entityname -- entity ) creates a new entity instance
func opNewEntity(state dtrules.State) error {
	nameObj, err := state.DataPop()
	if err != nil {
		return err
	}
	name, err := nameObj.RNameValue()
	if err != nil {
		return err
	}

	entity, err := state.GetSession().GetEntityFactory().CreateEntity(state.GetSession(), name)
	if err != nil {
		return err
	}
	return state.DataPush(entity.(dtrules.Object))
}

// opEntityName: ( entity -- name ) returns the entity's name
func opEntityName(state dtrules.State) error {
	obj, err := state.DataPop()
	if err != nil {
		return err
	}
	entity, err := obj.REntityValue()
	if err != nil {
		return err
	}
	return state.DataPush(entity.GetName())
}

// opEntityID: ( entity -- id ) returns the entity's ID
func opEntityID(state dtrules.State) error {
	obj, err := state.DataPop()
	if err != nil {
		return err
	}
	entity, err := obj.REntityValue()
	if err != nil {
		return err
	}
	return state.DataPush(dtrules.GetRIntegerValueFromInt(entity.GetID()))
}

// opReq: ( entity1 entity2 -- boolean ) reference equality for entities
// Returns true if both arguments refer to the same entity instance
func opReq(state dtrules.State) error {
	obj2, err := state.DataPop()
	if err != nil {
		return err
	}
	obj1, err := state.DataPop()
	if err != nil {
		return err
	}

	// Handle null comparisons
	if obj1.Type() == dtrules.TypeNull || obj2.Type() == dtrules.TypeNull {
		// null req null is true, anything else req null is false
		result := obj1.Type() == dtrules.TypeNull && obj2.Type() == dtrules.TypeNull
		return state.DataPush(dtrules.GetRBoolean(result))
	}

	// Both must be entities for reference equality
	entity1, err1 := obj1.REntityValue()
	entity2, err2 := obj2.REntityValue()

	if err1 != nil || err2 != nil {
		// If either isn't an entity, they're not reference-equal
		return state.DataPush(dtrules.GetRBoolean(false))
	}

	// Compare by entity ID (unique instance identifier)
	result := entity1.GetID() == entity2.GetID()
	return state.DataPush(dtrules.GetRBoolean(result))
}

// opIsDefined: ( /name -- boolean ) checks if a name is defined on the entity stack
// Unlike normal name lookup, this does NOT fail if the name is not found.
// Returns true if the name can be found on any entity in the entity stack.
func opIsDefined(state dtrules.State) error {
	nameObj, err := state.DataPop()
	if err != nil {
		return err
	}
	name, err := nameObj.RNameValue()
	if err != nil {
		return err
	}

	// Try to find the name on the entity stack
	obj, _ := state.Find(name)
	// If obj is not nil, the name is defined
	return state.DataPush(dtrules.GetRBoolean(obj != nil))
}
