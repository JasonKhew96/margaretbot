// Code generated by SQLBoiler 4.19.5 (https://github.com/aarondl/sqlboiler). DO NOT EDIT.
// This file is meant to be re-generated in place and/or deleted at any time.

package models

import (
	"bytes"
	"context"
	"reflect"
	"testing"

	"github.com/aarondl/randomize"
	"github.com/aarondl/sqlboiler/v4/boil"
	"github.com/aarondl/sqlboiler/v4/queries"
	"github.com/aarondl/strmangle"
)

var (
	// Relationships sometimes use the reflection helper queries.Equal/queries.Assign
	// so force a package dependency in case they don't.
	_ = queries.Equal
)

func testSubscriptions(t *testing.T) {
	t.Parallel()

	query := Subscriptions()

	if query.Query == nil {
		t.Error("expected a query, got nothing")
	}
}

func testSubscriptionsDelete(t *testing.T) {
	t.Parallel()

	seed := randomize.NewSeed()
	var err error
	o := &Subscription{}
	if err = randomize.Struct(seed, o, subscriptionDBTypes, true, subscriptionColumnsWithDefault...); err != nil {
		t.Errorf("Unable to randomize Subscription struct: %s", err)
	}

	ctx := context.Background()
	tx := MustTx(boil.BeginTx(ctx, nil))
	defer func() { _ = tx.Rollback() }()
	if err = o.Insert(ctx, tx, boil.Infer()); err != nil {
		t.Error(err)
	}

	if rowsAff, err := o.Delete(ctx, tx); err != nil {
		t.Error(err)
	} else if rowsAff != 1 {
		t.Error("should only have deleted one row, but affected:", rowsAff)
	}

	count, err := Subscriptions().Count(ctx, tx)
	if err != nil {
		t.Error(err)
	}

	if count != 0 {
		t.Error("want zero records, got:", count)
	}
}

func testSubscriptionsQueryDeleteAll(t *testing.T) {
	t.Parallel()

	seed := randomize.NewSeed()
	var err error
	o := &Subscription{}
	if err = randomize.Struct(seed, o, subscriptionDBTypes, true, subscriptionColumnsWithDefault...); err != nil {
		t.Errorf("Unable to randomize Subscription struct: %s", err)
	}

	ctx := context.Background()
	tx := MustTx(boil.BeginTx(ctx, nil))
	defer func() { _ = tx.Rollback() }()
	if err = o.Insert(ctx, tx, boil.Infer()); err != nil {
		t.Error(err)
	}

	if rowsAff, err := Subscriptions().DeleteAll(ctx, tx); err != nil {
		t.Error(err)
	} else if rowsAff != 1 {
		t.Error("should only have deleted one row, but affected:", rowsAff)
	}

	count, err := Subscriptions().Count(ctx, tx)
	if err != nil {
		t.Error(err)
	}

	if count != 0 {
		t.Error("want zero records, got:", count)
	}
}

func testSubscriptionsSliceDeleteAll(t *testing.T) {
	t.Parallel()

	seed := randomize.NewSeed()
	var err error
	o := &Subscription{}
	if err = randomize.Struct(seed, o, subscriptionDBTypes, true, subscriptionColumnsWithDefault...); err != nil {
		t.Errorf("Unable to randomize Subscription struct: %s", err)
	}

	ctx := context.Background()
	tx := MustTx(boil.BeginTx(ctx, nil))
	defer func() { _ = tx.Rollback() }()
	if err = o.Insert(ctx, tx, boil.Infer()); err != nil {
		t.Error(err)
	}

	slice := SubscriptionSlice{o}

	if rowsAff, err := slice.DeleteAll(ctx, tx); err != nil {
		t.Error(err)
	} else if rowsAff != 1 {
		t.Error("should only have deleted one row, but affected:", rowsAff)
	}

	count, err := Subscriptions().Count(ctx, tx)
	if err != nil {
		t.Error(err)
	}

	if count != 0 {
		t.Error("want zero records, got:", count)
	}
}

func testSubscriptionsExists(t *testing.T) {
	t.Parallel()

	seed := randomize.NewSeed()
	var err error
	o := &Subscription{}
	if err = randomize.Struct(seed, o, subscriptionDBTypes, true, subscriptionColumnsWithDefault...); err != nil {
		t.Errorf("Unable to randomize Subscription struct: %s", err)
	}

	ctx := context.Background()
	tx := MustTx(boil.BeginTx(ctx, nil))
	defer func() { _ = tx.Rollback() }()
	if err = o.Insert(ctx, tx, boil.Infer()); err != nil {
		t.Error(err)
	}

	e, err := SubscriptionExists(ctx, tx, o.ID)
	if err != nil {
		t.Errorf("Unable to check if Subscription exists: %s", err)
	}
	if !e {
		t.Errorf("Expected SubscriptionExists to return true, but got false.")
	}
}

func testSubscriptionsFind(t *testing.T) {
	t.Parallel()

	seed := randomize.NewSeed()
	var err error
	o := &Subscription{}
	if err = randomize.Struct(seed, o, subscriptionDBTypes, true, subscriptionColumnsWithDefault...); err != nil {
		t.Errorf("Unable to randomize Subscription struct: %s", err)
	}

	ctx := context.Background()
	tx := MustTx(boil.BeginTx(ctx, nil))
	defer func() { _ = tx.Rollback() }()
	if err = o.Insert(ctx, tx, boil.Infer()); err != nil {
		t.Error(err)
	}

	subscriptionFound, err := FindSubscription(ctx, tx, o.ID)
	if err != nil {
		t.Error(err)
	}

	if subscriptionFound == nil {
		t.Error("want a record, got nil")
	}
}

func testSubscriptionsBind(t *testing.T) {
	t.Parallel()

	seed := randomize.NewSeed()
	var err error
	o := &Subscription{}
	if err = randomize.Struct(seed, o, subscriptionDBTypes, true, subscriptionColumnsWithDefault...); err != nil {
		t.Errorf("Unable to randomize Subscription struct: %s", err)
	}

	ctx := context.Background()
	tx := MustTx(boil.BeginTx(ctx, nil))
	defer func() { _ = tx.Rollback() }()
	if err = o.Insert(ctx, tx, boil.Infer()); err != nil {
		t.Error(err)
	}

	if err = Subscriptions().Bind(ctx, tx, o); err != nil {
		t.Error(err)
	}
}

func testSubscriptionsOne(t *testing.T) {
	t.Parallel()

	seed := randomize.NewSeed()
	var err error
	o := &Subscription{}
	if err = randomize.Struct(seed, o, subscriptionDBTypes, true, subscriptionColumnsWithDefault...); err != nil {
		t.Errorf("Unable to randomize Subscription struct: %s", err)
	}

	ctx := context.Background()
	tx := MustTx(boil.BeginTx(ctx, nil))
	defer func() { _ = tx.Rollback() }()
	if err = o.Insert(ctx, tx, boil.Infer()); err != nil {
		t.Error(err)
	}

	if x, err := Subscriptions().One(ctx, tx); err != nil {
		t.Error(err)
	} else if x == nil {
		t.Error("expected to get a non nil record")
	}
}

func testSubscriptionsAll(t *testing.T) {
	t.Parallel()

	seed := randomize.NewSeed()
	var err error
	subscriptionOne := &Subscription{}
	subscriptionTwo := &Subscription{}
	if err = randomize.Struct(seed, subscriptionOne, subscriptionDBTypes, false, subscriptionColumnsWithDefault...); err != nil {
		t.Errorf("Unable to randomize Subscription struct: %s", err)
	}
	if err = randomize.Struct(seed, subscriptionTwo, subscriptionDBTypes, false, subscriptionColumnsWithDefault...); err != nil {
		t.Errorf("Unable to randomize Subscription struct: %s", err)
	}

	ctx := context.Background()
	tx := MustTx(boil.BeginTx(ctx, nil))
	defer func() { _ = tx.Rollback() }()
	if err = subscriptionOne.Insert(ctx, tx, boil.Infer()); err != nil {
		t.Error(err)
	}
	if err = subscriptionTwo.Insert(ctx, tx, boil.Infer()); err != nil {
		t.Error(err)
	}

	slice, err := Subscriptions().All(ctx, tx)
	if err != nil {
		t.Error(err)
	}

	if len(slice) != 2 {
		t.Error("want 2 records, got:", len(slice))
	}
}

func testSubscriptionsCount(t *testing.T) {
	t.Parallel()

	var err error
	seed := randomize.NewSeed()
	subscriptionOne := &Subscription{}
	subscriptionTwo := &Subscription{}
	if err = randomize.Struct(seed, subscriptionOne, subscriptionDBTypes, false, subscriptionColumnsWithDefault...); err != nil {
		t.Errorf("Unable to randomize Subscription struct: %s", err)
	}
	if err = randomize.Struct(seed, subscriptionTwo, subscriptionDBTypes, false, subscriptionColumnsWithDefault...); err != nil {
		t.Errorf("Unable to randomize Subscription struct: %s", err)
	}

	ctx := context.Background()
	tx := MustTx(boil.BeginTx(ctx, nil))
	defer func() { _ = tx.Rollback() }()
	if err = subscriptionOne.Insert(ctx, tx, boil.Infer()); err != nil {
		t.Error(err)
	}
	if err = subscriptionTwo.Insert(ctx, tx, boil.Infer()); err != nil {
		t.Error(err)
	}

	count, err := Subscriptions().Count(ctx, tx)
	if err != nil {
		t.Error(err)
	}

	if count != 2 {
		t.Error("want 2 records, got:", count)
	}
}

func subscriptionBeforeInsertHook(ctx context.Context, e boil.ContextExecutor, o *Subscription) error {
	*o = Subscription{}
	return nil
}

func subscriptionAfterInsertHook(ctx context.Context, e boil.ContextExecutor, o *Subscription) error {
	*o = Subscription{}
	return nil
}

func subscriptionAfterSelectHook(ctx context.Context, e boil.ContextExecutor, o *Subscription) error {
	*o = Subscription{}
	return nil
}

func subscriptionBeforeUpdateHook(ctx context.Context, e boil.ContextExecutor, o *Subscription) error {
	*o = Subscription{}
	return nil
}

func subscriptionAfterUpdateHook(ctx context.Context, e boil.ContextExecutor, o *Subscription) error {
	*o = Subscription{}
	return nil
}

func subscriptionBeforeDeleteHook(ctx context.Context, e boil.ContextExecutor, o *Subscription) error {
	*o = Subscription{}
	return nil
}

func subscriptionAfterDeleteHook(ctx context.Context, e boil.ContextExecutor, o *Subscription) error {
	*o = Subscription{}
	return nil
}

func subscriptionBeforeUpsertHook(ctx context.Context, e boil.ContextExecutor, o *Subscription) error {
	*o = Subscription{}
	return nil
}

func subscriptionAfterUpsertHook(ctx context.Context, e boil.ContextExecutor, o *Subscription) error {
	*o = Subscription{}
	return nil
}

func testSubscriptionsHooks(t *testing.T) {
	t.Parallel()

	var err error

	ctx := context.Background()
	empty := &Subscription{}
	o := &Subscription{}

	seed := randomize.NewSeed()
	if err = randomize.Struct(seed, o, subscriptionDBTypes, false); err != nil {
		t.Errorf("Unable to randomize Subscription object: %s", err)
	}

	AddSubscriptionHook(boil.BeforeInsertHook, subscriptionBeforeInsertHook)
	if err = o.doBeforeInsertHooks(ctx, nil); err != nil {
		t.Errorf("Unable to execute doBeforeInsertHooks: %s", err)
	}
	if !reflect.DeepEqual(o, empty) {
		t.Errorf("Expected BeforeInsertHook function to empty object, but got: %#v", o)
	}
	subscriptionBeforeInsertHooks = []SubscriptionHook{}

	AddSubscriptionHook(boil.AfterInsertHook, subscriptionAfterInsertHook)
	if err = o.doAfterInsertHooks(ctx, nil); err != nil {
		t.Errorf("Unable to execute doAfterInsertHooks: %s", err)
	}
	if !reflect.DeepEqual(o, empty) {
		t.Errorf("Expected AfterInsertHook function to empty object, but got: %#v", o)
	}
	subscriptionAfterInsertHooks = []SubscriptionHook{}

	AddSubscriptionHook(boil.AfterSelectHook, subscriptionAfterSelectHook)
	if err = o.doAfterSelectHooks(ctx, nil); err != nil {
		t.Errorf("Unable to execute doAfterSelectHooks: %s", err)
	}
	if !reflect.DeepEqual(o, empty) {
		t.Errorf("Expected AfterSelectHook function to empty object, but got: %#v", o)
	}
	subscriptionAfterSelectHooks = []SubscriptionHook{}

	AddSubscriptionHook(boil.BeforeUpdateHook, subscriptionBeforeUpdateHook)
	if err = o.doBeforeUpdateHooks(ctx, nil); err != nil {
		t.Errorf("Unable to execute doBeforeUpdateHooks: %s", err)
	}
	if !reflect.DeepEqual(o, empty) {
		t.Errorf("Expected BeforeUpdateHook function to empty object, but got: %#v", o)
	}
	subscriptionBeforeUpdateHooks = []SubscriptionHook{}

	AddSubscriptionHook(boil.AfterUpdateHook, subscriptionAfterUpdateHook)
	if err = o.doAfterUpdateHooks(ctx, nil); err != nil {
		t.Errorf("Unable to execute doAfterUpdateHooks: %s", err)
	}
	if !reflect.DeepEqual(o, empty) {
		t.Errorf("Expected AfterUpdateHook function to empty object, but got: %#v", o)
	}
	subscriptionAfterUpdateHooks = []SubscriptionHook{}

	AddSubscriptionHook(boil.BeforeDeleteHook, subscriptionBeforeDeleteHook)
	if err = o.doBeforeDeleteHooks(ctx, nil); err != nil {
		t.Errorf("Unable to execute doBeforeDeleteHooks: %s", err)
	}
	if !reflect.DeepEqual(o, empty) {
		t.Errorf("Expected BeforeDeleteHook function to empty object, but got: %#v", o)
	}
	subscriptionBeforeDeleteHooks = []SubscriptionHook{}

	AddSubscriptionHook(boil.AfterDeleteHook, subscriptionAfterDeleteHook)
	if err = o.doAfterDeleteHooks(ctx, nil); err != nil {
		t.Errorf("Unable to execute doAfterDeleteHooks: %s", err)
	}
	if !reflect.DeepEqual(o, empty) {
		t.Errorf("Expected AfterDeleteHook function to empty object, but got: %#v", o)
	}
	subscriptionAfterDeleteHooks = []SubscriptionHook{}

	AddSubscriptionHook(boil.BeforeUpsertHook, subscriptionBeforeUpsertHook)
	if err = o.doBeforeUpsertHooks(ctx, nil); err != nil {
		t.Errorf("Unable to execute doBeforeUpsertHooks: %s", err)
	}
	if !reflect.DeepEqual(o, empty) {
		t.Errorf("Expected BeforeUpsertHook function to empty object, but got: %#v", o)
	}
	subscriptionBeforeUpsertHooks = []SubscriptionHook{}

	AddSubscriptionHook(boil.AfterUpsertHook, subscriptionAfterUpsertHook)
	if err = o.doAfterUpsertHooks(ctx, nil); err != nil {
		t.Errorf("Unable to execute doAfterUpsertHooks: %s", err)
	}
	if !reflect.DeepEqual(o, empty) {
		t.Errorf("Expected AfterUpsertHook function to empty object, but got: %#v", o)
	}
	subscriptionAfterUpsertHooks = []SubscriptionHook{}
}

func testSubscriptionsInsert(t *testing.T) {
	t.Parallel()

	seed := randomize.NewSeed()
	var err error
	o := &Subscription{}
	if err = randomize.Struct(seed, o, subscriptionDBTypes, true, subscriptionColumnsWithDefault...); err != nil {
		t.Errorf("Unable to randomize Subscription struct: %s", err)
	}

	ctx := context.Background()
	tx := MustTx(boil.BeginTx(ctx, nil))
	defer func() { _ = tx.Rollback() }()
	if err = o.Insert(ctx, tx, boil.Infer()); err != nil {
		t.Error(err)
	}

	count, err := Subscriptions().Count(ctx, tx)
	if err != nil {
		t.Error(err)
	}

	if count != 1 {
		t.Error("want one record, got:", count)
	}
}

func testSubscriptionsInsertWhitelist(t *testing.T) {
	t.Parallel()

	seed := randomize.NewSeed()
	var err error
	o := &Subscription{}
	if err = randomize.Struct(seed, o, subscriptionDBTypes, true); err != nil {
		t.Errorf("Unable to randomize Subscription struct: %s", err)
	}

	ctx := context.Background()
	tx := MustTx(boil.BeginTx(ctx, nil))
	defer func() { _ = tx.Rollback() }()
	if err = o.Insert(ctx, tx, boil.Whitelist(strmangle.SetMerge(subscriptionPrimaryKeyColumns, subscriptionColumnsWithoutDefault)...)); err != nil {
		t.Error(err)
	}

	count, err := Subscriptions().Count(ctx, tx)
	if err != nil {
		t.Error(err)
	}

	if count != 1 {
		t.Error("want one record, got:", count)
	}
}

func testSubscriptionsReload(t *testing.T) {
	t.Parallel()

	seed := randomize.NewSeed()
	var err error
	o := &Subscription{}
	if err = randomize.Struct(seed, o, subscriptionDBTypes, true, subscriptionColumnsWithDefault...); err != nil {
		t.Errorf("Unable to randomize Subscription struct: %s", err)
	}

	ctx := context.Background()
	tx := MustTx(boil.BeginTx(ctx, nil))
	defer func() { _ = tx.Rollback() }()
	if err = o.Insert(ctx, tx, boil.Infer()); err != nil {
		t.Error(err)
	}

	if err = o.Reload(ctx, tx); err != nil {
		t.Error(err)
	}
}

func testSubscriptionsReloadAll(t *testing.T) {
	t.Parallel()

	seed := randomize.NewSeed()
	var err error
	o := &Subscription{}
	if err = randomize.Struct(seed, o, subscriptionDBTypes, true, subscriptionColumnsWithDefault...); err != nil {
		t.Errorf("Unable to randomize Subscription struct: %s", err)
	}

	ctx := context.Background()
	tx := MustTx(boil.BeginTx(ctx, nil))
	defer func() { _ = tx.Rollback() }()
	if err = o.Insert(ctx, tx, boil.Infer()); err != nil {
		t.Error(err)
	}

	slice := SubscriptionSlice{o}

	if err = slice.ReloadAll(ctx, tx); err != nil {
		t.Error(err)
	}
}

func testSubscriptionsSelect(t *testing.T) {
	t.Parallel()

	seed := randomize.NewSeed()
	var err error
	o := &Subscription{}
	if err = randomize.Struct(seed, o, subscriptionDBTypes, true, subscriptionColumnsWithDefault...); err != nil {
		t.Errorf("Unable to randomize Subscription struct: %s", err)
	}

	ctx := context.Background()
	tx := MustTx(boil.BeginTx(ctx, nil))
	defer func() { _ = tx.Rollback() }()
	if err = o.Insert(ctx, tx, boil.Infer()); err != nil {
		t.Error(err)
	}

	slice, err := Subscriptions().All(ctx, tx)
	if err != nil {
		t.Error(err)
	}

	if len(slice) != 1 {
		t.Error("want one record, got:", len(slice))
	}
}

var (
	subscriptionDBTypes = map[string]string{`ID`: `INTEGER`, `ChannelID`: `TEXT`, `ThreadID`: `INTEGER`, `Regex`: `TEXT`, `RegexBan`: `TEXT`, `ExpiredAt`: `TIMESTAMP`, `CreatedAt`: `TIMESTAMP`, `UpdatedAt`: `TIMESTAMP`}
	_                   = bytes.MinRead
)

func testSubscriptionsUpdate(t *testing.T) {
	t.Parallel()

	if 0 == len(subscriptionPrimaryKeyColumns) {
		t.Skip("Skipping table with no primary key columns")
	}
	if len(subscriptionAllColumns) == len(subscriptionPrimaryKeyColumns) {
		t.Skip("Skipping table with only primary key columns")
	}

	seed := randomize.NewSeed()
	var err error
	o := &Subscription{}
	if err = randomize.Struct(seed, o, subscriptionDBTypes, true, subscriptionColumnsWithDefault...); err != nil {
		t.Errorf("Unable to randomize Subscription struct: %s", err)
	}

	ctx := context.Background()
	tx := MustTx(boil.BeginTx(ctx, nil))
	defer func() { _ = tx.Rollback() }()
	if err = o.Insert(ctx, tx, boil.Infer()); err != nil {
		t.Error(err)
	}

	count, err := Subscriptions().Count(ctx, tx)
	if err != nil {
		t.Error(err)
	}

	if count != 1 {
		t.Error("want one record, got:", count)
	}

	if err = randomize.Struct(seed, o, subscriptionDBTypes, true, subscriptionPrimaryKeyColumns...); err != nil {
		t.Errorf("Unable to randomize Subscription struct: %s", err)
	}

	if rowsAff, err := o.Update(ctx, tx, boil.Infer()); err != nil {
		t.Error(err)
	} else if rowsAff != 1 {
		t.Error("should only affect one row but affected", rowsAff)
	}
}

func testSubscriptionsSliceUpdateAll(t *testing.T) {
	t.Parallel()

	if len(subscriptionAllColumns) == len(subscriptionPrimaryKeyColumns) {
		t.Skip("Skipping table with only primary key columns")
	}

	seed := randomize.NewSeed()
	var err error
	o := &Subscription{}
	if err = randomize.Struct(seed, o, subscriptionDBTypes, true, subscriptionColumnsWithDefault...); err != nil {
		t.Errorf("Unable to randomize Subscription struct: %s", err)
	}

	ctx := context.Background()
	tx := MustTx(boil.BeginTx(ctx, nil))
	defer func() { _ = tx.Rollback() }()
	if err = o.Insert(ctx, tx, boil.Infer()); err != nil {
		t.Error(err)
	}

	count, err := Subscriptions().Count(ctx, tx)
	if err != nil {
		t.Error(err)
	}

	if count != 1 {
		t.Error("want one record, got:", count)
	}

	if err = randomize.Struct(seed, o, subscriptionDBTypes, true, subscriptionPrimaryKeyColumns...); err != nil {
		t.Errorf("Unable to randomize Subscription struct: %s", err)
	}

	// Remove Primary keys and unique columns from what we plan to update
	var fields []string
	if strmangle.StringSliceMatch(subscriptionAllColumns, subscriptionPrimaryKeyColumns) {
		fields = subscriptionAllColumns
	} else {
		fields = strmangle.SetComplement(
			subscriptionAllColumns,
			subscriptionPrimaryKeyColumns,
		)
		fields = strmangle.SetComplement(fields, subscriptionGeneratedColumns)
	}

	value := reflect.Indirect(reflect.ValueOf(o))
	typ := reflect.TypeOf(o).Elem()
	n := typ.NumField()

	updateMap := M{}
	for _, col := range fields {
		for i := 0; i < n; i++ {
			f := typ.Field(i)
			if f.Tag.Get("boil") == col {
				updateMap[col] = value.Field(i).Interface()
			}
		}
	}

	slice := SubscriptionSlice{o}
	if rowsAff, err := slice.UpdateAll(ctx, tx, updateMap); err != nil {
		t.Error(err)
	} else if rowsAff != 1 {
		t.Error("wanted one record updated but got", rowsAff)
	}
}

func testSubscriptionsUpsert(t *testing.T) {
	t.Parallel()
	if len(subscriptionAllColumns) == len(subscriptionPrimaryKeyColumns) {
		t.Skip("Skipping table with only primary key columns")
	}

	seed := randomize.NewSeed()
	var err error
	// Attempt the INSERT side of an UPSERT
	o := Subscription{}
	if err = randomize.Struct(seed, &o, subscriptionDBTypes, true); err != nil {
		t.Errorf("Unable to randomize Subscription struct: %s", err)
	}

	ctx := context.Background()
	tx := MustTx(boil.BeginTx(ctx, nil))
	defer func() { _ = tx.Rollback() }()
	if err = o.Upsert(ctx, tx, false, nil, boil.Infer(), boil.Infer()); err != nil {
		t.Errorf("Unable to upsert Subscription: %s", err)
	}

	count, err := Subscriptions().Count(ctx, tx)
	if err != nil {
		t.Error(err)
	}
	if count != 1 {
		t.Error("want one record, got:", count)
	}

	// Attempt the UPDATE side of an UPSERT
	if err = randomize.Struct(seed, &o, subscriptionDBTypes, false, subscriptionPrimaryKeyColumns...); err != nil {
		t.Errorf("Unable to randomize Subscription struct: %s", err)
	}

	if err = o.Upsert(ctx, tx, true, nil, boil.Infer(), boil.Infer()); err != nil {
		t.Errorf("Unable to upsert Subscription: %s", err)
	}

	count, err = Subscriptions().Count(ctx, tx)
	if err != nil {
		t.Error(err)
	}
	if count != 1 {
		t.Error("want one record, got:", count)
	}
}
