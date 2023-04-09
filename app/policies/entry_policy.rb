# typed: strict
# frozen_string_literal: true

class EntryPolicy < ApplicationPolicy
  sig { returns(T::Boolean) }
  def destroy?
    profile == T.cast(record, Entry).profile
  end
end
