# typed: true

# DO NOT EDIT MANUALLY
# This is an autogenerated file for dynamic methods in `Account`.
# Please instead update this file by running `bin/tapioca dsl Account`.

class Account
  include GeneratedAssociationMethods
  include GeneratedAttributeMethods
  extend CommonRelationMethods
  extend GeneratedRelationMethods

  private

  sig { returns(NilClass) }
  def to_ary; end

  module CommonRelationMethods
    sig { params(block: T.nilable(T.proc.params(record: ::Account).returns(T.untyped))).returns(T::Boolean) }
    def any?(&block); end

    sig { params(column_name: T.any(String, Symbol)).returns(T.untyped) }
    def average(column_name); end

    sig { params(attributes: T.untyped, block: T.nilable(T.proc.params(object: ::Account).void)).returns(::Account) }
    def build(attributes = nil, &block); end

    sig { params(operation: Symbol, column_name: T.any(String, Symbol)).returns(T.untyped) }
    def calculate(operation, column_name); end

    sig { params(column_name: T.untyped).returns(T.untyped) }
    def count(column_name = nil); end

    sig { params(attributes: T.untyped, block: T.nilable(T.proc.params(object: ::Account).void)).returns(::Account) }
    def create(attributes = nil, &block); end

    sig { params(attributes: T.untyped, block: T.nilable(T.proc.params(object: ::Account).void)).returns(::Account) }
    def create!(attributes = nil, &block); end

    sig { params(attributes: T.untyped, block: T.nilable(T.proc.params(object: ::Account).void)).returns(::Account) }
    def create_or_find_by(attributes, &block); end

    sig { params(attributes: T.untyped, block: T.nilable(T.proc.params(object: ::Account).void)).returns(::Account) }
    def create_or_find_by!(attributes, &block); end

    sig { returns(T::Array[::Account]) }
    def destroy_all; end

    sig { params(conditions: T.untyped).returns(T::Boolean) }
    def exists?(conditions = :none); end

    sig { returns(T.nilable(::Account)) }
    def fifth; end

    sig { returns(::Account) }
    def fifth!; end

    sig { params(args: T.untyped).returns(T.untyped) }
    def find(*args); end

    sig { params(args: T.untyped).returns(T.nilable(::Account)) }
    def find_by(*args); end

    sig { params(args: T.untyped).returns(::Account) }
    def find_by!(*args); end

    sig { params(attributes: T.untyped, block: T.nilable(T.proc.params(object: ::Account).void)).returns(::Account) }
    def find_or_create_by(attributes, &block); end

    sig { params(attributes: T.untyped, block: T.nilable(T.proc.params(object: ::Account).void)).returns(::Account) }
    def find_or_create_by!(attributes, &block); end

    sig { params(attributes: T.untyped, block: T.nilable(T.proc.params(object: ::Account).void)).returns(::Account) }
    def find_or_initialize_by(attributes, &block); end

    sig { params(signed_id: T.untyped, purpose: T.untyped).returns(T.nilable(::Account)) }
    def find_signed(signed_id, purpose: nil); end

    sig { params(signed_id: T.untyped, purpose: T.untyped).returns(::Account) }
    def find_signed!(signed_id, purpose: nil); end

    sig { params(arg: T.untyped, args: T.untyped).returns(::Account) }
    def find_sole_by(arg, *args); end

    sig { params(limit: T.untyped).returns(T.untyped) }
    def first(limit = nil); end

    sig { returns(::Account) }
    def first!; end

    sig { returns(T.nilable(::Account)) }
    def forty_two; end

    sig { returns(::Account) }
    def forty_two!; end

    sig { returns(T.nilable(::Account)) }
    def fourth; end

    sig { returns(::Account) }
    def fourth!; end

    sig { returns(Array) }
    def ids; end

    sig { params(record: T.untyped).returns(T::Boolean) }
    def include?(record); end

    sig { params(limit: T.untyped).returns(T.untyped) }
    def last(limit = nil); end

    sig { returns(::Account) }
    def last!; end

    sig { params(block: T.nilable(T.proc.params(record: ::Account).returns(T.untyped))).returns(T::Boolean) }
    def many?(&block); end

    sig { params(column_name: T.any(String, Symbol)).returns(T.untyped) }
    def maximum(column_name); end

    sig { params(record: T.untyped).returns(T::Boolean) }
    def member?(record); end

    sig { params(column_name: T.any(String, Symbol)).returns(T.untyped) }
    def minimum(column_name); end

    sig { params(attributes: T.untyped, block: T.nilable(T.proc.params(object: ::Account).void)).returns(::Account) }
    def new(attributes = nil, &block); end

    sig { params(block: T.nilable(T.proc.params(record: ::Account).returns(T.untyped))).returns(T::Boolean) }
    def none?(&block); end

    sig { params(block: T.nilable(T.proc.params(record: ::Account).returns(T.untyped))).returns(T::Boolean) }
    def one?(&block); end

    sig { params(column_names: T.untyped).returns(T.untyped) }
    def pick(*column_names); end

    sig { params(column_names: T.untyped).returns(T.untyped) }
    def pluck(*column_names); end

    sig { returns(T.nilable(::Account)) }
    def second; end

    sig { returns(::Account) }
    def second!; end

    sig { returns(T.nilable(::Account)) }
    def second_to_last; end

    sig { returns(::Account) }
    def second_to_last!; end

    sig { returns(::Account) }
    def sole; end

    sig do
      params(
        column_name: T.nilable(T.any(String, Symbol)),
        block: T.nilable(T.proc.params(record: T.untyped).returns(T.untyped))
      ).returns(T.untyped)
    end
    def sum(column_name = nil, &block); end

    sig { params(limit: T.untyped).returns(T.untyped) }
    def take(limit = nil); end

    sig { returns(::Account) }
    def take!; end

    sig { returns(T.nilable(::Account)) }
    def third; end

    sig { returns(::Account) }
    def third!; end

    sig { returns(T.nilable(::Account)) }
    def third_to_last; end

    sig { returns(::Account) }
    def third_to_last!; end
  end

  module GeneratedAssociationMethods
    sig { returns(T.nilable(::AccountProfile)) }
    def account_profile; end

    sig { params(value: T.nilable(::AccountProfile)).void }
    def account_profile=(value); end

    sig { params(args: T.untyped, blk: T.untyped).returns(::AccountProfile) }
    def build_account_profile(*args, &blk); end

    sig { params(args: T.untyped, blk: T.untyped).returns(::Profile) }
    def build_profile(*args, &blk); end

    sig { params(args: T.untyped, blk: T.untyped).returns(::AccountProfile) }
    def create_account_profile(*args, &blk); end

    sig { params(args: T.untyped, blk: T.untyped).returns(::AccountProfile) }
    def create_account_profile!(*args, &blk); end

    sig { params(args: T.untyped, blk: T.untyped).returns(::Profile) }
    def create_profile(*args, &blk); end

    sig { params(args: T.untyped, blk: T.untyped).returns(::Profile) }
    def create_profile!(*args, &blk); end

    sig { returns(T.nilable(::Profile)) }
    def profile; end

    sig { params(value: T.nilable(::Profile)).void }
    def profile=(value); end

    sig { returns(T.nilable(::AccountProfile)) }
    def reload_account_profile; end

    sig { returns(T.nilable(::Profile)) }
    def reload_profile; end
  end

  module GeneratedAssociationRelationMethods
    sig { returns(PrivateAssociationRelation) }
    def all; end

    sig { params(args: T.untyped, blk: T.untyped).returns(PrivateAssociationRelation) }
    def and(*args, &blk); end

    sig { params(args: T.untyped, blk: T.untyped).returns(PrivateAssociationRelation) }
    def annotate(*args, &blk); end

    sig { params(args: T.untyped, blk: T.untyped).returns(PrivateAssociationRelation) }
    def create_with(*args, &blk); end

    sig { params(args: T.untyped, blk: T.untyped).returns(PrivateAssociationRelation) }
    def distinct(*args, &blk); end

    sig { params(args: T.untyped, blk: T.untyped).returns(PrivateAssociationRelation) }
    def eager_load(*args, &blk); end

    sig { params(args: T.untyped, blk: T.untyped).returns(PrivateAssociationRelation) }
    def except(*args, &blk); end

    sig { params(args: T.untyped, blk: T.untyped).returns(PrivateAssociationRelation) }
    def excluding(*args, &blk); end

    sig { params(args: T.untyped, blk: T.untyped).returns(PrivateAssociationRelation) }
    def extending(*args, &blk); end

    sig { params(args: T.untyped, blk: T.untyped).returns(PrivateAssociationRelation) }
    def extract_associated(*args, &blk); end

    sig { params(args: T.untyped, blk: T.untyped).returns(PrivateAssociationRelation) }
    def from(*args, &blk); end

    sig { params(args: T.untyped, blk: T.untyped).returns(PrivateAssociationRelation) }
    def group(*args, &blk); end

    sig { params(args: T.untyped, blk: T.untyped).returns(PrivateAssociationRelation) }
    def having(*args, &blk); end

    sig { params(args: T.untyped, blk: T.untyped).returns(PrivateAssociationRelation) }
    def in_order_of(*args, &blk); end

    sig { params(args: T.untyped, blk: T.untyped).returns(PrivateAssociationRelation) }
    def includes(*args, &blk); end

    sig do
      params(
        attributes: Hash,
        returning: T.nilable(T.any(T::Array[Symbol], FalseClass)),
        unique_by: T.nilable(T.any(T::Array[Symbol], Symbol))
      ).returns(ActiveRecord::Result)
    end
    def insert(attributes, returning: nil, unique_by: nil); end

    sig do
      params(
        attributes: Hash,
        returning: T.nilable(T.any(T::Array[Symbol], FalseClass))
      ).returns(ActiveRecord::Result)
    end
    def insert!(attributes, returning: nil); end

    sig do
      params(
        attributes: T::Array[Hash],
        returning: T.nilable(T.any(T::Array[Symbol], FalseClass)),
        unique_by: T.nilable(T.any(T::Array[Symbol], Symbol))
      ).returns(ActiveRecord::Result)
    end
    def insert_all(attributes, returning: nil, unique_by: nil); end

    sig do
      params(
        attributes: T::Array[Hash],
        returning: T.nilable(T.any(T::Array[Symbol], FalseClass))
      ).returns(ActiveRecord::Result)
    end
    def insert_all!(attributes, returning: nil); end

    sig { params(args: T.untyped, blk: T.untyped).returns(PrivateAssociationRelation) }
    def invert_where(*args, &blk); end

    sig { params(args: T.untyped, blk: T.untyped).returns(PrivateAssociationRelation) }
    def joins(*args, &blk); end

    sig { params(args: T.untyped, blk: T.untyped).returns(PrivateAssociationRelation) }
    def left_joins(*args, &blk); end

    sig { params(args: T.untyped, blk: T.untyped).returns(PrivateAssociationRelation) }
    def left_outer_joins(*args, &blk); end

    sig { params(args: T.untyped, blk: T.untyped).returns(PrivateAssociationRelation) }
    def limit(*args, &blk); end

    sig { params(args: T.untyped, blk: T.untyped).returns(PrivateAssociationRelation) }
    def lock(*args, &blk); end

    sig { params(args: T.untyped, blk: T.untyped).returns(PrivateAssociationRelation) }
    def merge(*args, &blk); end

    sig { params(args: T.untyped, blk: T.untyped).returns(PrivateAssociationRelation) }
    def none(*args, &blk); end

    sig { params(args: T.untyped, blk: T.untyped).returns(PrivateAssociationRelation) }
    def offset(*args, &blk); end

    sig { params(args: T.untyped, blk: T.untyped).returns(PrivateAssociationRelation) }
    def only(*args, &blk); end

    sig { params(args: T.untyped, blk: T.untyped).returns(PrivateAssociationRelation) }
    def optimizer_hints(*args, &blk); end

    sig { params(args: T.untyped, blk: T.untyped).returns(PrivateAssociationRelation) }
    def or(*args, &blk); end

    sig { params(args: T.untyped, blk: T.untyped).returns(PrivateAssociationRelation) }
    def order(*args, &blk); end

    sig { params(args: T.untyped, blk: T.untyped).returns(PrivateAssociationRelation) }
    def preload(*args, &blk); end

    sig { params(args: T.untyped, blk: T.untyped).returns(PrivateAssociationRelation) }
    def readonly(*args, &blk); end

    sig { params(args: T.untyped, blk: T.untyped).returns(PrivateAssociationRelation) }
    def references(*args, &blk); end

    sig { params(args: T.untyped, blk: T.untyped).returns(PrivateAssociationRelation) }
    def reorder(*args, &blk); end

    sig { params(args: T.untyped, blk: T.untyped).returns(PrivateAssociationRelation) }
    def reselect(*args, &blk); end

    sig { params(args: T.untyped, blk: T.untyped).returns(PrivateAssociationRelation) }
    def reverse_order(*args, &blk); end

    sig { params(args: T.untyped, blk: T.untyped).returns(PrivateAssociationRelation) }
    def rewhere(*args, &blk); end

    sig { params(args: T.untyped, blk: T.untyped).returns(PrivateAssociationRelation) }
    def select(*args, &blk); end

    sig { params(args: T.untyped, blk: T.untyped).returns(PrivateAssociationRelation) }
    def strict_loading(*args, &blk); end

    sig { params(args: T.untyped, blk: T.untyped).returns(PrivateAssociationRelation) }
    def structurally_compatible?(*args, &blk); end

    sig { params(args: T.untyped, blk: T.untyped).returns(PrivateAssociationRelation) }
    def uniq!(*args, &blk); end

    sig { params(args: T.untyped, blk: T.untyped).returns(PrivateAssociationRelation) }
    def unscope(*args, &blk); end

    sig do
      params(
        attributes: Hash,
        returning: T.nilable(T.any(T::Array[Symbol], FalseClass)),
        unique_by: T.nilable(T.any(T::Array[Symbol], Symbol))
      ).returns(ActiveRecord::Result)
    end
    def upsert(attributes, returning: nil, unique_by: nil); end

    sig do
      params(
        attributes: T::Array[Hash],
        returning: T.nilable(T.any(T::Array[Symbol], FalseClass)),
        unique_by: T.nilable(T.any(T::Array[Symbol], Symbol))
      ).returns(ActiveRecord::Result)
    end
    def upsert_all(attributes, returning: nil, unique_by: nil); end

    sig { params(args: T.untyped, blk: T.untyped).returns(PrivateAssociationRelationWhereChain) }
    def where(*args, &blk); end

    sig { params(args: T.untyped, blk: T.untyped).returns(PrivateAssociationRelation) }
    def without(*args, &blk); end
  end

  module GeneratedAttributeMethods
    sig { returns(T.nilable(::ActiveSupport::TimeWithZone)) }
    def created_at; end

    sig { params(value: ::ActiveSupport::TimeWithZone).returns(::ActiveSupport::TimeWithZone) }
    def created_at=(value); end

    sig { returns(T::Boolean) }
    def created_at?; end

    sig { returns(T.nilable(::ActiveSupport::TimeWithZone)) }
    def created_at_before_last_save; end

    sig { returns(T.untyped) }
    def created_at_before_type_cast; end

    sig { returns(T::Boolean) }
    def created_at_came_from_user?; end

    sig { returns(T.nilable([T.nilable(::ActiveSupport::TimeWithZone), T.nilable(::ActiveSupport::TimeWithZone)])) }
    def created_at_change; end

    sig { returns(T.nilable([T.nilable(::ActiveSupport::TimeWithZone), T.nilable(::ActiveSupport::TimeWithZone)])) }
    def created_at_change_to_be_saved; end

    sig { returns(T::Boolean) }
    def created_at_changed?; end

    sig { returns(T.nilable(::ActiveSupport::TimeWithZone)) }
    def created_at_in_database; end

    sig { returns(T.nilable([T.nilable(::ActiveSupport::TimeWithZone), T.nilable(::ActiveSupport::TimeWithZone)])) }
    def created_at_previous_change; end

    sig { returns(T::Boolean) }
    def created_at_previously_changed?; end

    sig { returns(T.nilable(::ActiveSupport::TimeWithZone)) }
    def created_at_previously_was; end

    sig { returns(T.nilable(::ActiveSupport::TimeWithZone)) }
    def created_at_was; end

    sig { void }
    def created_at_will_change!; end

    sig { returns(T.nilable(::ActiveSupport::TimeWithZone)) }
    def current_signed_in_at; end

    sig { params(value: T.nilable(::ActiveSupport::TimeWithZone)).returns(T.nilable(::ActiveSupport::TimeWithZone)) }
    def current_signed_in_at=(value); end

    sig { returns(T::Boolean) }
    def current_signed_in_at?; end

    sig { returns(T.nilable(::ActiveSupport::TimeWithZone)) }
    def current_signed_in_at_before_last_save; end

    sig { returns(T.untyped) }
    def current_signed_in_at_before_type_cast; end

    sig { returns(T::Boolean) }
    def current_signed_in_at_came_from_user?; end

    sig { returns(T.nilable([T.nilable(::ActiveSupport::TimeWithZone), T.nilable(::ActiveSupport::TimeWithZone)])) }
    def current_signed_in_at_change; end

    sig { returns(T.nilable([T.nilable(::ActiveSupport::TimeWithZone), T.nilable(::ActiveSupport::TimeWithZone)])) }
    def current_signed_in_at_change_to_be_saved; end

    sig { returns(T::Boolean) }
    def current_signed_in_at_changed?; end

    sig { returns(T.nilable(::ActiveSupport::TimeWithZone)) }
    def current_signed_in_at_in_database; end

    sig { returns(T.nilable([T.nilable(::ActiveSupport::TimeWithZone), T.nilable(::ActiveSupport::TimeWithZone)])) }
    def current_signed_in_at_previous_change; end

    sig { returns(T::Boolean) }
    def current_signed_in_at_previously_changed?; end

    sig { returns(T.nilable(::ActiveSupport::TimeWithZone)) }
    def current_signed_in_at_previously_was; end

    sig { returns(T.nilable(::ActiveSupport::TimeWithZone)) }
    def current_signed_in_at_was; end

    sig { void }
    def current_signed_in_at_will_change!; end

    sig { returns(T.untyped) }
    def id; end

    sig { params(value: T.untyped).returns(T.untyped) }
    def id=(value); end

    sig { returns(T::Boolean) }
    def id?; end

    sig { returns(T.untyped) }
    def id_before_last_save; end

    sig { returns(T.untyped) }
    def id_before_type_cast; end

    sig { returns(T::Boolean) }
    def id_came_from_user?; end

    sig { returns(T.nilable([T.untyped, T.untyped])) }
    def id_change; end

    sig { returns(T.nilable([T.untyped, T.untyped])) }
    def id_change_to_be_saved; end

    sig { returns(T::Boolean) }
    def id_changed?; end

    sig { returns(T.untyped) }
    def id_in_database; end

    sig { returns(T.nilable([T.untyped, T.untyped])) }
    def id_previous_change; end

    sig { returns(T::Boolean) }
    def id_previously_changed?; end

    sig { returns(T.untyped) }
    def id_previously_was; end

    sig { returns(T.untyped) }
    def id_was; end

    sig { void }
    def id_will_change!; end

    sig { returns(T.nilable(::ActiveSupport::TimeWithZone)) }
    def last_signed_in_at; end

    sig { params(value: T.nilable(::ActiveSupport::TimeWithZone)).returns(T.nilable(::ActiveSupport::TimeWithZone)) }
    def last_signed_in_at=(value); end

    sig { returns(T::Boolean) }
    def last_signed_in_at?; end

    sig { returns(T.nilable(::ActiveSupport::TimeWithZone)) }
    def last_signed_in_at_before_last_save; end

    sig { returns(T.untyped) }
    def last_signed_in_at_before_type_cast; end

    sig { returns(T::Boolean) }
    def last_signed_in_at_came_from_user?; end

    sig { returns(T.nilable([T.nilable(::ActiveSupport::TimeWithZone), T.nilable(::ActiveSupport::TimeWithZone)])) }
    def last_signed_in_at_change; end

    sig { returns(T.nilable([T.nilable(::ActiveSupport::TimeWithZone), T.nilable(::ActiveSupport::TimeWithZone)])) }
    def last_signed_in_at_change_to_be_saved; end

    sig { returns(T::Boolean) }
    def last_signed_in_at_changed?; end

    sig { returns(T.nilable(::ActiveSupport::TimeWithZone)) }
    def last_signed_in_at_in_database; end

    sig { returns(T.nilable([T.nilable(::ActiveSupport::TimeWithZone), T.nilable(::ActiveSupport::TimeWithZone)])) }
    def last_signed_in_at_previous_change; end

    sig { returns(T::Boolean) }
    def last_signed_in_at_previously_changed?; end

    sig { returns(T.nilable(::ActiveSupport::TimeWithZone)) }
    def last_signed_in_at_previously_was; end

    sig { returns(T.nilable(::ActiveSupport::TimeWithZone)) }
    def last_signed_in_at_was; end

    sig { void }
    def last_signed_in_at_will_change!; end

    sig { returns(T.untyped) }
    def locale; end

    sig { params(value: T.untyped).returns(T.untyped) }
    def locale=(value); end

    sig { returns(T::Boolean) }
    def locale?; end

    sig { returns(T.untyped) }
    def locale_before_last_save; end

    sig { returns(T.untyped) }
    def locale_before_type_cast; end

    sig { returns(T::Boolean) }
    def locale_came_from_user?; end

    sig { returns(T.nilable([T.untyped, T.untyped])) }
    def locale_change; end

    sig { returns(T.nilable([T.untyped, T.untyped])) }
    def locale_change_to_be_saved; end

    sig { returns(T::Boolean) }
    def locale_changed?; end

    sig { returns(T.untyped) }
    def locale_in_database; end

    sig { returns(T.nilable([T.untyped, T.untyped])) }
    def locale_previous_change; end

    sig { returns(T::Boolean) }
    def locale_previously_changed?; end

    sig { returns(T.untyped) }
    def locale_previously_was; end

    sig { returns(T.untyped) }
    def locale_was; end

    sig { void }
    def locale_will_change!; end

    sig { returns(::String) }
    def phone_number; end

    sig { params(value: ::String).returns(::String) }
    def phone_number=(value); end

    sig { returns(T::Boolean) }
    def phone_number?; end

    sig { returns(T.nilable(::String)) }
    def phone_number_before_last_save; end

    sig { returns(T.untyped) }
    def phone_number_before_type_cast; end

    sig { returns(T::Boolean) }
    def phone_number_came_from_user?; end

    sig { returns(T.nilable([::String, ::String])) }
    def phone_number_change; end

    sig { returns(T.nilable([::String, ::String])) }
    def phone_number_change_to_be_saved; end

    sig { returns(T::Boolean) }
    def phone_number_changed?; end

    sig { returns(T.nilable(::String)) }
    def phone_number_in_database; end

    sig { returns(T.nilable([::String, ::String])) }
    def phone_number_previous_change; end

    sig { returns(T::Boolean) }
    def phone_number_previously_changed?; end

    sig { returns(T.nilable(::String)) }
    def phone_number_previously_was; end

    sig { returns(T.nilable(::String)) }
    def phone_number_was; end

    sig { void }
    def phone_number_will_change!; end

    sig { void }
    def restore_created_at!; end

    sig { void }
    def restore_current_signed_in_at!; end

    sig { void }
    def restore_id!; end

    sig { void }
    def restore_last_signed_in_at!; end

    sig { void }
    def restore_locale!; end

    sig { void }
    def restore_phone_number!; end

    sig { void }
    def restore_sign_in_count!; end

    sig { void }
    def restore_updated_at!; end

    sig { returns(T.nilable([T.nilable(::ActiveSupport::TimeWithZone), T.nilable(::ActiveSupport::TimeWithZone)])) }
    def saved_change_to_created_at; end

    sig { returns(T::Boolean) }
    def saved_change_to_created_at?; end

    sig { returns(T.nilable([T.nilable(::ActiveSupport::TimeWithZone), T.nilable(::ActiveSupport::TimeWithZone)])) }
    def saved_change_to_current_signed_in_at; end

    sig { returns(T::Boolean) }
    def saved_change_to_current_signed_in_at?; end

    sig { returns(T.nilable([T.untyped, T.untyped])) }
    def saved_change_to_id; end

    sig { returns(T::Boolean) }
    def saved_change_to_id?; end

    sig { returns(T.nilable([T.nilable(::ActiveSupport::TimeWithZone), T.nilable(::ActiveSupport::TimeWithZone)])) }
    def saved_change_to_last_signed_in_at; end

    sig { returns(T::Boolean) }
    def saved_change_to_last_signed_in_at?; end

    sig { returns(T.nilable([T.untyped, T.untyped])) }
    def saved_change_to_locale; end

    sig { returns(T::Boolean) }
    def saved_change_to_locale?; end

    sig { returns(T.nilable([::String, ::String])) }
    def saved_change_to_phone_number; end

    sig { returns(T::Boolean) }
    def saved_change_to_phone_number?; end

    sig { returns(T.nilable([::Integer, ::Integer])) }
    def saved_change_to_sign_in_count; end

    sig { returns(T::Boolean) }
    def saved_change_to_sign_in_count?; end

    sig { returns(T.nilable([T.nilable(::ActiveSupport::TimeWithZone), T.nilable(::ActiveSupport::TimeWithZone)])) }
    def saved_change_to_updated_at; end

    sig { returns(T::Boolean) }
    def saved_change_to_updated_at?; end

    sig { returns(::Integer) }
    def sign_in_count; end

    sig { params(value: ::Integer).returns(::Integer) }
    def sign_in_count=(value); end

    sig { returns(T::Boolean) }
    def sign_in_count?; end

    sig { returns(T.nilable(::Integer)) }
    def sign_in_count_before_last_save; end

    sig { returns(T.untyped) }
    def sign_in_count_before_type_cast; end

    sig { returns(T::Boolean) }
    def sign_in_count_came_from_user?; end

    sig { returns(T.nilable([::Integer, ::Integer])) }
    def sign_in_count_change; end

    sig { returns(T.nilable([::Integer, ::Integer])) }
    def sign_in_count_change_to_be_saved; end

    sig { returns(T::Boolean) }
    def sign_in_count_changed?; end

    sig { returns(T.nilable(::Integer)) }
    def sign_in_count_in_database; end

    sig { returns(T.nilable([::Integer, ::Integer])) }
    def sign_in_count_previous_change; end

    sig { returns(T::Boolean) }
    def sign_in_count_previously_changed?; end

    sig { returns(T.nilable(::Integer)) }
    def sign_in_count_previously_was; end

    sig { returns(T.nilable(::Integer)) }
    def sign_in_count_was; end

    sig { void }
    def sign_in_count_will_change!; end

    sig { returns(T.nilable(::ActiveSupport::TimeWithZone)) }
    def updated_at; end

    sig { params(value: ::ActiveSupport::TimeWithZone).returns(::ActiveSupport::TimeWithZone) }
    def updated_at=(value); end

    sig { returns(T::Boolean) }
    def updated_at?; end

    sig { returns(T.nilable(::ActiveSupport::TimeWithZone)) }
    def updated_at_before_last_save; end

    sig { returns(T.untyped) }
    def updated_at_before_type_cast; end

    sig { returns(T::Boolean) }
    def updated_at_came_from_user?; end

    sig { returns(T.nilable([T.nilable(::ActiveSupport::TimeWithZone), T.nilable(::ActiveSupport::TimeWithZone)])) }
    def updated_at_change; end

    sig { returns(T.nilable([T.nilable(::ActiveSupport::TimeWithZone), T.nilable(::ActiveSupport::TimeWithZone)])) }
    def updated_at_change_to_be_saved; end

    sig { returns(T::Boolean) }
    def updated_at_changed?; end

    sig { returns(T.nilable(::ActiveSupport::TimeWithZone)) }
    def updated_at_in_database; end

    sig { returns(T.nilable([T.nilable(::ActiveSupport::TimeWithZone), T.nilable(::ActiveSupport::TimeWithZone)])) }
    def updated_at_previous_change; end

    sig { returns(T::Boolean) }
    def updated_at_previously_changed?; end

    sig { returns(T.nilable(::ActiveSupport::TimeWithZone)) }
    def updated_at_previously_was; end

    sig { returns(T.nilable(::ActiveSupport::TimeWithZone)) }
    def updated_at_was; end

    sig { void }
    def updated_at_will_change!; end

    sig { returns(T::Boolean) }
    def will_save_change_to_created_at?; end

    sig { returns(T::Boolean) }
    def will_save_change_to_current_signed_in_at?; end

    sig { returns(T::Boolean) }
    def will_save_change_to_id?; end

    sig { returns(T::Boolean) }
    def will_save_change_to_last_signed_in_at?; end

    sig { returns(T::Boolean) }
    def will_save_change_to_locale?; end

    sig { returns(T::Boolean) }
    def will_save_change_to_phone_number?; end

    sig { returns(T::Boolean) }
    def will_save_change_to_sign_in_count?; end

    sig { returns(T::Boolean) }
    def will_save_change_to_updated_at?; end
  end

  module GeneratedRelationMethods
    sig { returns(PrivateRelation) }
    def all; end

    sig { params(args: T.untyped, blk: T.untyped).returns(PrivateRelation) }
    def and(*args, &blk); end

    sig { params(args: T.untyped, blk: T.untyped).returns(PrivateRelation) }
    def annotate(*args, &blk); end

    sig { params(args: T.untyped, blk: T.untyped).returns(PrivateRelation) }
    def create_with(*args, &blk); end

    sig { params(args: T.untyped, blk: T.untyped).returns(PrivateRelation) }
    def distinct(*args, &blk); end

    sig { params(args: T.untyped, blk: T.untyped).returns(PrivateRelation) }
    def eager_load(*args, &blk); end

    sig { params(args: T.untyped, blk: T.untyped).returns(PrivateRelation) }
    def except(*args, &blk); end

    sig { params(args: T.untyped, blk: T.untyped).returns(PrivateRelation) }
    def excluding(*args, &blk); end

    sig { params(args: T.untyped, blk: T.untyped).returns(PrivateRelation) }
    def extending(*args, &blk); end

    sig { params(args: T.untyped, blk: T.untyped).returns(PrivateRelation) }
    def extract_associated(*args, &blk); end

    sig { params(args: T.untyped, blk: T.untyped).returns(PrivateRelation) }
    def from(*args, &blk); end

    sig { params(args: T.untyped, blk: T.untyped).returns(PrivateRelation) }
    def group(*args, &blk); end

    sig { params(args: T.untyped, blk: T.untyped).returns(PrivateRelation) }
    def having(*args, &blk); end

    sig { params(args: T.untyped, blk: T.untyped).returns(PrivateRelation) }
    def in_order_of(*args, &blk); end

    sig { params(args: T.untyped, blk: T.untyped).returns(PrivateRelation) }
    def includes(*args, &blk); end

    sig { params(args: T.untyped, blk: T.untyped).returns(PrivateRelation) }
    def invert_where(*args, &blk); end

    sig { params(args: T.untyped, blk: T.untyped).returns(PrivateRelation) }
    def joins(*args, &blk); end

    sig { params(args: T.untyped, blk: T.untyped).returns(PrivateRelation) }
    def left_joins(*args, &blk); end

    sig { params(args: T.untyped, blk: T.untyped).returns(PrivateRelation) }
    def left_outer_joins(*args, &blk); end

    sig { params(args: T.untyped, blk: T.untyped).returns(PrivateRelation) }
    def limit(*args, &blk); end

    sig { params(args: T.untyped, blk: T.untyped).returns(PrivateRelation) }
    def lock(*args, &blk); end

    sig { params(args: T.untyped, blk: T.untyped).returns(PrivateRelation) }
    def merge(*args, &blk); end

    sig { params(args: T.untyped, blk: T.untyped).returns(PrivateRelation) }
    def none(*args, &blk); end

    sig { params(args: T.untyped, blk: T.untyped).returns(PrivateRelation) }
    def offset(*args, &blk); end

    sig { params(args: T.untyped, blk: T.untyped).returns(PrivateRelation) }
    def only(*args, &blk); end

    sig { params(args: T.untyped, blk: T.untyped).returns(PrivateRelation) }
    def optimizer_hints(*args, &blk); end

    sig { params(args: T.untyped, blk: T.untyped).returns(PrivateRelation) }
    def or(*args, &blk); end

    sig { params(args: T.untyped, blk: T.untyped).returns(PrivateRelation) }
    def order(*args, &blk); end

    sig { params(args: T.untyped, blk: T.untyped).returns(PrivateRelation) }
    def preload(*args, &blk); end

    sig { params(args: T.untyped, blk: T.untyped).returns(PrivateRelation) }
    def readonly(*args, &blk); end

    sig { params(args: T.untyped, blk: T.untyped).returns(PrivateRelation) }
    def references(*args, &blk); end

    sig { params(args: T.untyped, blk: T.untyped).returns(PrivateRelation) }
    def reorder(*args, &blk); end

    sig { params(args: T.untyped, blk: T.untyped).returns(PrivateRelation) }
    def reselect(*args, &blk); end

    sig { params(args: T.untyped, blk: T.untyped).returns(PrivateRelation) }
    def reverse_order(*args, &blk); end

    sig { params(args: T.untyped, blk: T.untyped).returns(PrivateRelation) }
    def rewhere(*args, &blk); end

    sig { params(args: T.untyped, blk: T.untyped).returns(PrivateRelation) }
    def select(*args, &blk); end

    sig { params(args: T.untyped, blk: T.untyped).returns(PrivateRelation) }
    def strict_loading(*args, &blk); end

    sig { params(args: T.untyped, blk: T.untyped).returns(PrivateRelation) }
    def structurally_compatible?(*args, &blk); end

    sig { params(args: T.untyped, blk: T.untyped).returns(PrivateRelation) }
    def uniq!(*args, &blk); end

    sig { params(args: T.untyped, blk: T.untyped).returns(PrivateRelation) }
    def unscope(*args, &blk); end

    sig { params(args: T.untyped, blk: T.untyped).returns(PrivateRelationWhereChain) }
    def where(*args, &blk); end

    sig { params(args: T.untyped, blk: T.untyped).returns(PrivateRelation) }
    def without(*args, &blk); end
  end

  class PrivateAssociationRelation < ::ActiveRecord::AssociationRelation
    include CommonRelationMethods
    include GeneratedAssociationRelationMethods

    Elem = type_member { { fixed: ::Account } }

    sig { returns(T::Array[::Account]) }
    def to_ary; end
  end

  class PrivateAssociationRelationWhereChain < PrivateAssociationRelation
    Elem = type_member { { fixed: ::Account } }

    sig { params(args: T.untyped).returns(PrivateAssociationRelation) }
    def associated(*args); end

    sig { params(args: T.untyped).returns(PrivateAssociationRelation) }
    def missing(*args); end

    sig { params(opts: T.untyped, rest: T.untyped).returns(PrivateAssociationRelation) }
    def not(opts, *rest); end
  end

  class PrivateCollectionProxy < ::ActiveRecord::Associations::CollectionProxy
    include CommonRelationMethods
    include GeneratedAssociationRelationMethods

    Elem = type_member { { fixed: ::Account } }

    sig do
      params(
        records: T.any(::Account, T::Enumerable[T.any(::Account, T::Enumerable[::Account])])
      ).returns(PrivateCollectionProxy)
    end
    def <<(*records); end

    sig do
      params(
        records: T.any(::Account, T::Enumerable[T.any(::Account, T::Enumerable[::Account])])
      ).returns(PrivateCollectionProxy)
    end
    def append(*records); end

    sig { returns(PrivateCollectionProxy) }
    def clear; end

    sig do
      params(
        records: T.any(::Account, T::Enumerable[T.any(::Account, T::Enumerable[::Account])])
      ).returns(PrivateCollectionProxy)
    end
    def concat(*records); end

    sig do
      params(
        records: T.any(::Account, Integer, String, T::Enumerable[T.any(::Account, Integer, String, T::Enumerable[::Account])])
      ).returns(T::Array[::Account])
    end
    def delete(*records); end

    sig do
      params(
        records: T.any(::Account, Integer, String, T::Enumerable[T.any(::Account, Integer, String, T::Enumerable[::Account])])
      ).returns(T::Array[::Account])
    end
    def destroy(*records); end

    sig { returns(T::Array[::Account]) }
    def load_target; end

    sig do
      params(
        records: T.any(::Account, T::Enumerable[T.any(::Account, T::Enumerable[::Account])])
      ).returns(PrivateCollectionProxy)
    end
    def prepend(*records); end

    sig do
      params(
        records: T.any(::Account, T::Enumerable[T.any(::Account, T::Enumerable[::Account])])
      ).returns(PrivateCollectionProxy)
    end
    def push(*records); end

    sig do
      params(
        other_array: T.any(::Account, T::Enumerable[T.any(::Account, T::Enumerable[::Account])])
      ).returns(T::Array[::Account])
    end
    def replace(other_array); end

    sig { returns(PrivateAssociationRelation) }
    def scope; end

    sig { returns(T::Array[::Account]) }
    def target; end

    sig { returns(T::Array[::Account]) }
    def to_ary; end
  end

  class PrivateRelation < ::ActiveRecord::Relation
    include CommonRelationMethods
    include GeneratedRelationMethods

    Elem = type_member { { fixed: ::Account } }

    sig { returns(T::Array[::Account]) }
    def to_ary; end
  end

  class PrivateRelationWhereChain < PrivateRelation
    Elem = type_member { { fixed: ::Account } }

    sig { params(args: T.untyped).returns(PrivateRelation) }
    def associated(*args); end

    sig { params(args: T.untyped).returns(PrivateRelation) }
    def missing(*args); end

    sig { params(opts: T.untyped, rest: T.untyped).returns(PrivateRelation) }
    def not(opts, *rest); end
  end
end
