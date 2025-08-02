# typed: strict
# frozen_string_literal: true

class PostLinkRecord < ApplicationRecord
  self.table_name = "post_links"

  belongs_to :post_record, class_name: "PostRecord", foreign_key: :post_id
  belongs_to :link_record, class_name: "LinkRecord", foreign_key: :link_id
end
