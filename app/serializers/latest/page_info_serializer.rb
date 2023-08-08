# typed: false
# frozen_string_literal: true

class Latest::PageInfoSerializer < Latest::ApplicationSerializer
  attributes :end_cursor
  attributes :has_next_page
  attributes :has_previous_page
  attributes :start_cursor
end
