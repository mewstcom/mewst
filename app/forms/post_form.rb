# typed: strict
# frozen_string_literal: true

class PostForm < ApplicationForm
  attribute :body, :string

  attr_accessor :profile

  validates :body, presence: true
  validates :profile, presence: true
end
