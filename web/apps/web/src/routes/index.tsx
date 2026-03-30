import { createFileRoute, Navigate } from '@tanstack/react-router'

export const Route = createFileRoute('/')({
	component: () => <Navigate to="/playground" search={{ screen: '', state: '', set: 'wireframe', layout: '', width: 0 }} />,
})
