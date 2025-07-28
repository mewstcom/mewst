# typed: strict
# frozen_string_literal: true

class PostLinkRecord < ApplicationRecord
  self.table_name = "post_links"

  belongs_to :post
  belongs_to :link
end