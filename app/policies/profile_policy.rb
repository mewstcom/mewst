# typed: strict
# frozen_string_literal: true

class ProfilePolicy < ApplicationPolicy
  sig { returns(T::Boolean) }
  def show?
    profile == T.cast(record, Post).profile
  end
end
