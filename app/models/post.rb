# typed: strict
# frozen_string_literal: true

class Post < ApplicationRecord
  belongs_to :profile
end
