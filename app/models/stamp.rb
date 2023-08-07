# typed: strict
# frozen_string_literal: true

class Stamp < ApplicationRecord
  counter_culture :comment_post

  belongs_to :comment_post
  belongs_to :profile
end
