# typed: strict
# frozen_string_literal: true

class ProfilePolicy < ApplicationPolicy
  sig { returns(T::Boolean) }
  def show?
    profile == T.cast(record, ProfileRecord)
  end
end
