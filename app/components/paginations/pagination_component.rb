# typed: strict
# frozen_string_literal: true

class Paginations::PaginationComponent < ApplicationComponent
  sig { params(page_info: PageInfo, previous_path: String, next_path: String).void }
  def initialize(page_info:, previous_path:, next_path:)
    @page_info = T.let(page_info, PageInfo)
    @previous_path = T.let(previous_path, String)
    @next_path = T.let(next_path, String)
  end

  sig { returns(PageInfo) }
  attr_reader :page_info
  private :page_info

  sig { returns(String) }
  attr_reader :previous_path
  private :previous_path

  sig { returns(String) }
  attr_reader :next_path
  private :next_path
end
