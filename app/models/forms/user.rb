# typed: strict
# frozen_string_literal: true

class Forms::User < Forms::Base
  attribute :locale, :string

  sig { returns(User) }
  attr_accessor :user

  validates :locale, presence: true
end
