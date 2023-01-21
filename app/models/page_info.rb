# typed: strict
# frozen_string_literal: true

class PageInfo < T::Struct
  const :end_cursor, T.nilable(String)
  const :has_next_page, T::Boolean
  const :has_previous_page, T::Boolean
  const :start_cursor, T.nilable(String)
end
