# typed: false
# frozen_string_literal: true

class V1::PageInfoSerializer < V1::ApplicationSerializer
  root_key :page_info

  attributes :end_cursor
  attributes :has_next_page
  attributes :has_previous_page
  attributes :start_cursor
end
