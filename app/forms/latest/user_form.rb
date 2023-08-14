# typed: strict
# frozen_string_literal: true

class Latest::UserForm < Latest::ApplicationForm
  attribute :locale, :string

  sig { returns(T.nilable(User)) }
  attr_accessor :user

  validates :locale, presence: true
end
