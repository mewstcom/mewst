# typed: strict
# frozen_string_literal: true

class PostLink < ApplicationRecord
  belongs_to :post
  belongs_to :link
end
