# typed: strict
# frozen_string_literal: true

class Paginations::CursorPaginationComponent < ApplicationComponent
  sig { params(prev_path: String, next_path: String, page_info: PageInfo).void }
  def initialize(prev_path:, next_path:, page_info:)
    @prev_path = T.let(prev_path, String)
    @next_path = T.let(next_path, String)
    @page_info = T.let(page_info, PageInfo)
  end
end
